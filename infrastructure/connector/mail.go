package connector

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"sqldb-ws/domain/domain_service"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/utils"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

type EmailData struct {
	Name string
	Code string
}

type CachedMail struct {
	From          string
	To            string
	MailRecord    utils.Record
	IsValidButton bool
	Timestamp     time.Time
}

func ForgeMail(from utils.Record, to utils.Record, subject string, tpl string,
	bodyToMap map[string]interface{}, domain utils.DomainITF, tplID int64,
	bodySchema int64, destID int64, destOnResponse int64, fileAttached string, signature string) (utils.Record, error) {
	var subj bytes.Buffer
	var content bytes.Buffer
	tmplSubj, err := template.New("email").Parse(subject)
	if err == nil {
		if err := tmplSubj.Execute(&subj, bodyToMap); err == nil {
			subject = subj.String()
		}
	}
	tmpl, err := template.New("email").Parse(tpl + "<br>" + signature)
	if err != nil {
		return utils.Record{}, err
	}

	if err := tmpl.Execute(&content, bodyToMap); err != nil {
		return utils.Record{}, err
	}
	m := utils.Record{
		"from_email": utils.GetString(from, "id"),
		"to_email": []interface{}{map[string]interface{}{
			utils.SpecialIDParam: to[utils.SpecialIDParam],
			"name":               to["name"],
		}}, // SHOULD BE ID
		"subject":               strings.ReplaceAll(strings.ReplaceAll(subject, "é", "e"), "é", "e"),
		"content":               strings.ReplaceAll(strings.ReplaceAll(content.String(), "''", "'"), "''", "'"),
		"file_attached":         fileAttached,
		ds.EmailTemplateDBField: tplID,
	}
	if destOnResponse > -1 {
		m[ds.DestTableDBField+"_on_response"] = destOnResponse
	}
	if bodySchema > -1 {
		m["mapped_with"+ds.SchemaDBField] = bodySchema
	}
	if destID > -1 {
		m["mapped_with"+ds.DestTableDBField] = destID
	}
	if m["code"] == nil || m["code"] == "" {
		m["code"] = uuid.New()
	}
	return m, nil
}

func formatSubject(subject string) string {
	// Use Q-encoding to make the subject RFC 2047 compliant
	return mime.QEncoding.Encode("utf-8", subject)
}

// ---------- Detect if SMTP is unreachable ----------
func isSMTPUnreachable(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok {
		return !ne.Temporary() || ne.Timeout()
	}
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "no such host") {
		return true
	}
	return false
}

// ---------- Retry wrapper ----------
func SendMailWithRetry(from, to string, mail utils.Record, isValidButton bool, maxRetries int, cacheDir string) error {
	backoff := 2 * time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cache, err := sendMail(from, to, mail, isValidButton)
		if err == nil {
			fileName := fmt.Sprintf("%d_%s.json", time.Now().UnixNano(), strings.ReplaceAll(cache.To, "@", "_"))
			os.Remove(filepath.Join(cacheDir, fileName))
			return nil
		}
		if !isSMTPUnreachable(err) {
			return err // non-retryable
		}
		fmt.Printf("Attempt %d/%d failed, SMTP unreachable: %v\n", attempt, maxRetries, err)
		time.Sleep(backoff)
		backoff *= 2
	}
	return fmt.Errorf("all retries failed for %s", to)
}

// ---------- Cache local ----------
func cacheMail(cm CachedMail, cacheDir string) error {
	os.MkdirAll(cacheDir, 0755)
	fileName := fmt.Sprintf("%d_%s.json", time.Now().UnixNano(), strings.ReplaceAll(cm.To, "@", "_"))
	data, err := json.Marshal(cm)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, fileName), data, 0644)
}

// ---------- Send safe with cache ----------
func SendMailSafe(from, to string, mail utils.Record, isValidButton bool) error {
	cacheDir := "./mail_cache"
	err := SendMailWithRetry(from, to, mail, isValidButton, 3, cacheDir)
	if err != nil && isSMTPUnreachable(err) {
		fmt.Println("SMTP unreachable, caching mail:", to)
		cm := CachedMail{From: from, To: to, MailRecord: mail, IsValidButton: isValidButton, Timestamp: time.Now()}
		return cacheMail(cm, cacheDir)
	}
	return err
}

// ---------- SendMail with MIME ----------
func sendMail(from, to string, mail utils.Record, isValidButton bool) (CachedMail, error) {
	var body bytes.Buffer
	boundary := "mixed-boundary"
	altboundary := "alt-boundary"

	writeLine := func(s string) {
		body.WriteString(s + "\r\n")
	}

	// Headers
	writeLine(fmt.Sprintf("From: %s", from))
	writeLine(fmt.Sprintf("To: %s", to))
	writeLine("Subject: " + formatSubject(utils.GetString(mail, "subject")))
	writeLine("MIME-Version: 1.0")
	writeLine("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"")
	writeLine("")

	// Multipart alternative (HTML)
	writeLine("--" + boundary)
	writeLine("Content-Type: multipart/alternative; boundary=\"" + altboundary + "\"")
	writeLine("")

	writeLine("--" + altboundary)
	writeLine("Content-Type: text/html; charset=\"utf-8\"")
	writeLine("Content-Transfer-Encoding: 7bit")
	writeLine("")
	writeLine("<html><head><meta charset=\"UTF-8\"></head>")
	writeLine("<body style=\"margin:0; padding:0; font-family:Arial, sans-serif;\">")
	writeLine(utils.GetString(mail, "content"))

	code := utils.GetString(mail, "code")
	if isValidButton {
		host := os.Getenv("HOST")
		if host == "" {
			host = ""
		}
		writeLine(fmt.Sprintf(`
			<table role="presentation" cellspacing="0" cellpadding="0" border="0" align="center">
				<tr>
					<td align="center" valign="middle" style="padding-right:10px;">
					<table role="presentation" cellspacing="0" cellpadding="0" border="0">
						<tr>
						<td bgcolor="#13aa52" style="border-radius:5px;">
							<a href="%s/v1/response/%s?got_response=true"
							target="_blank"
							style="display:inline-block; padding:12px 18px; font-size:18px; font-family:Helvetica, Arial, sans-serif; color:#ffffff; text-decoration:none; font-weight:bold; border-radius:5px;">
							  <img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABgAAAAYCAYAAADgdz34AAAACXBIWXMAAAsTAAALEwEAmpwYAAABNElEQVR4nO2VvUoDQRSGv40iNyoK2kQDSyCwsg6io6CKFoIk8g0Ed3ClY2FpZ2v4BZQ8gVpIkSJsJAiJK0kpZ2aU2EUH1nJ3s9975szuzsLuDgNQ2i+I4QqoehbsAa9AZfAfXAHvAAlfACJvoFK2bK9gBN2AXhgHWpU3bPN7iP6p4HcDq2AH7h7N8ZL2xYypgmMUA5YAXaBw4A0MEPgGcA48Ay0xO4dBEqsZx6eS2kSmqbcf28Ac8A9yClgC1gBWxWUZ8y7A3wvxqALuCk1KwMZwGbK0tc1jFvMyJjNgaLrV0Hu6YAk8C3c5+B1M90DgJt5W3E8QCrhq2KhELrDrhoGRh1H4AJPJfmsHrbL/ytFnXOHqS6n9AxkOQYyHwQ7cAGGrEW3p1eqpAtqGxyqYi5dyfKeGpZRjvLj9ALjBBMn4g4+VpAAAAAElFTkSuQmCC" width="18" height="18">
							</a>
						</td>
						</tr>
					</table>
					</td>

					<td align="center" valign="middle">
					<table role="presentation" cellspacing="0" cellpadding="0" border="0">
						<tr>
						<td bgcolor="#FF4742" style="border-radius:5px;">
							<a href="%s/v1/response/%s?got_response=false"
							target="_blank"
							style="display:inline-block; padding:12px 18px; font-size:18px; font-family:Helvetica, Arial, sans-serif; color:#ffffff; text-decoration:none; font-weight:bold; border-radius:5px;">
								<img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABgAAAAYCAYAAADgdz34AAAACXBIWXMAAAsTAAALEwEAmpwYAAABM0lEQVR4nO2VvUoDQRSGv40iNyoK2kQDSyCwsg6io6CKFoIk8g0Ed3ClY2FpZ2v4BZQ8gVpIkSJsJAiJK0kpZ2aU2EUH1nJ3s9975szuzsLuDgNQ2i+I4QqoehbsAa9AZfAfXAHvAAlfACJvoFK2bK9gBN2AXhgHWpU3bPN7iP6p4HcDq2AH7h7N8ZL2xYypgmMUA5YAXaBw4A0MEPgGcA48Ay0xO4dBEqsZx6eS2kSmqbcf28Ac8A9yClgC1gBWxWUZ8y7A3wvxqALuCk1KwMZwGbK0tc1jFvMyJjNgaLrV0Hu6YAk8C3c5+B1M90DgJt5W3E8QCrhq2KhELrDrhoGRh1H4AJPJfmsHrbL/ytFnXOHqS6n9AxkOQYyHwQ7cAGGrEW3p1eqpAtqGxyqYi5dyfKeGpZRjvLj9ALjBBMn4g4+VpAAAAAElFTkSuQmCC" width="18" height="18">
							</a>
						</td>
						</tr>
					</table>
					</td>
				</tr>
				</table><br>`, host, code, host, code))
	}

	writeLine("</body></html>")

	// End multipart alternative
	writeLine("--" + altboundary + "--")
	writeLine("")

	// Attachments
	if fileAttached := utils.GetString(mail, "file_attached"); fileAttached != "" {
		files := strings.Split(fileAttached, ",")
		for _, filePath := range files {
			filePath = strings.TrimSpace(filePath)
			fileName := filePath
			if strings.Contains(filePath, "/") {
				parts := strings.Split(filePath, "/")
				fileName = parts[len(parts)-1]
			}
			if !strings.HasPrefix(filePath, "/mnt/files/") {
				filePath = "/mnt/files/" + filePath
			}
			uncompPath, err := domain_service.UncompressGzip(filePath)
			if err != nil {
				fmt.Println("Could not uncompress file from :", uncompPath, uncompPath, err)
				continue
			}
			data, err := os.ReadFile(uncompPath)
			if err != nil {
				fmt.Println("Could not read file:", uncompPath, err)
				continue
			}
			domain_service.DeleteUncompressed(uncompPath)
			writeLine("--" + boundary)
			writeLine("Content-Type: application/octet-stream")
			writeLine("Content-Transfer-Encoding: base64")
			encodedName := mime.QEncoding.Encode("UTF-8", fileName)
			writeLine("Content-Disposition: attachment; filename=\"" + encodedName + "\"")
			writeLine("")

			encoded := base64.StdEncoding.EncodeToString(data)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				writeLine(encoded[i:end])
			}
		}
	}

	// End multipart/mixed
	writeLine("--" + boundary + "--")

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpPwd := os.Getenv("SMTP_PASSWORD")

	if smtpHost == "" || smtpPort == "" {
		return CachedMail{From: from, To: to, MailRecord: mail, IsValidButton: isValidButton, Timestamp: time.Now()}, fmt.Errorf("SMTP_HOST or SMTP_PORT not set")
	}

	var auth smtp.Auth
	if smtpPwd != "" {
		auth = smtp.PlainAuth("", from, smtpPwd, smtpHost)
	}

	return CachedMail{From: from, To: to, MailRecord: mail, IsValidButton: isValidButton, Timestamp: time.Now()}, smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, body.Bytes())
}
