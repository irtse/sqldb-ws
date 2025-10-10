package connector

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"net/smtp"
	"os"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/utils"
	"strings"
	"text/template"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type EmailData struct {
	Name string
	Code string
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

func SendMail(from string, to string, mail utils.Record, isValidButton bool) error {
	var body bytes.Buffer
	boundary := "mixed-boundary"
	altboundary := "alt-boundary"

	writeLine := func(s string) {
		body.WriteString(s + "\r\n")
	}

	// En-têtes
	writeLine(fmt.Sprintf("From: %s", from))
	writeLine(fmt.Sprintf("To: %s", to))
	writeLine("Subject: " + formatSubject(utils.GetString(mail, "subject")))
	writeLine("MIME-Version: 1.0")
	writeLine("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"")
	writeLine("")

	// Début multipart/mixed
	writeLine("--" + boundary)
	writeLine("Content-Type: multipart/alternative; boundary=\"" + altboundary + "\"")
	writeLine("")

	// Partie texte HTML
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
			host = "http://capitalisation.irt-aese.local"
		}
		writeLine(fmt.Sprintf(`
			<div style="display:flex;justify-content:center;align-items: center;">
				<br>
				<table border="0" cellspacing="0" cellpadding="0" style="margin:0 10px 0 0">
					<tr>
						<td align="center" style="border-radius: 5px; background-color: #13aa52;">
							<a rel="noopener" target="_blank" href="%s/v1/response/%s?got_response=true" style="font-size: 18px; font-family: Helvetica, Arial, sans-serif; color: #ffffff; font-weight: bold; text-decoration: none;border-radius: 5px; padding: 12px 18px; border: 1px solid #13aa52; display: inline-block;">✔</a>
						</td>
					</tr>
				</table>
				<table border="0" cellspacing="0" cellpadding="0">
					<tr>
						<td align="center" style="border-radius: 5px; background-color: #FF4742;">
							<a rel="noopener" target="_blank" href="%s/v1/response/%s?got_response=false" style="font-size: 18px; font-family: Helvetica, Arial, sans-serif; color: #ffffff; font-weight: bold; text-decoration: none;border-radius: 5px; padding: 12px 18px; border: 1px solid #FF4742; display: inline-block;">✘</a>
						</td>
					</tr>
				</table>
			</div>
			<br>`, host, code, host, code))
	}

	writeLine("</body></html>")

	// Fin du multipart/alternative
	writeLine("--" + altboundary + "--")
	writeLine("")

	// Pièces jointes
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

			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Println("Could not read file:", filePath, err)
				continue
			}

			writeLine("--" + boundary)
			writeLine("Content-Type: application/octet-stream")
			writeLine("Content-Transfer-Encoding: base64")

			// Encodage MIME pour le nom du fichier
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

	// Fin du multipart/mixed
	writeLine("--" + boundary + "--")

	// Envoi SMTP
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	pwd := os.Getenv("SMTP_PASSWORD")

	var err error
	if smtpHost == "" || smtpPort == "" {
		return fmt.Errorf("SMTP_HOST or SMTP_PORT not set")
	}

	if pwd != "" {
		auth := smtp.PlainAuth("", from, pwd, smtpHost)
		err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, body.Bytes())
	} else {
		err = smtp.SendMail(smtpHost+":"+smtpPort, nil, from, []string{to}, body.Bytes())
	}

	if err != nil {
		fmt.Println("EMAIL NOT SENT:", err)
		return err
	}

	fmt.Println("EMAIL SENT")
	return nil
}

// RemoveAccents transforms é → e, à → a, ç → c, etc.
func RemoveAccents(input string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, input)
	return result
}
