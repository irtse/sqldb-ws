package controllers

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"os"
	"sqldb-ws/controllers/controller"
	"sqldb-ws/domain"
	"sqldb-ws/domain/utils"

	"github.com/matthewhartstonge/argon2"
)

// Operations about login
type AuthController struct{ controller.AbstractController }

// LLDAP HERE
// func (l *AuthController) LoginLDAP() { }

var key = []byte("zpnbsswigxgnttgjqjlcnowoaishpqel") // 32 bytes
var iv = []byte("mhtwqevzehivjzjj")

// @Title Login
// @Description User login
// @Param	body		body 	Credential	true		"Credentials"
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @router /login [post]
func (l *AuthController) Login() {
	// login function will overide generic procedure foundable in controllers.go
	body := l.Body(false)             // extracting body
	if log, ok := body["login"]; ok { // search for login in body
		response, err := domain.IsLogged(false, utils.ToString(log), "")
		if err != nil {
			l.Response(response, err, "", "")
			return
		}
		if len(response) == 0 {
			l.Response(response, errors.New("AUTH : username/email invalid"), "", "")
			return
		}
		valid := false
		// if no problem check if logger is authorized to work on API and properly registered
		if os.Getenv("AUTH_MODE") == "ldap" && utils.GetString(response[0], "name") != "root" {
			plain, err := decrypt(utils.GetString(body, "password"), key, iv)
			if err != nil {
				l.Response(response, err, "", "")
				return
			}
			valid = controller.CheckLdap(utils.GetString(response[0], "name"), plain)
			if !valid {
				valid = controller.CheckLdap(utils.GetString(response[0], "name"), utils.GetString(body, "password"))
			}
		} else {
			pass, ok := body["password"] // then compare password founded in base and ... whatever... you know what's about
			plain, err := decrypt(utils.ToString(pass), key, iv)
			if err != nil {
				l.Response(response, err, "", "")
				return
			}
			pwd, ok1 := response[0]["password"]
			if ok && ok1 {
				valid, _ = argon2.VerifyEncoded([]byte(utils.ToString(plain)), []byte(utils.ToString(pwd)))
				if !valid {
					valid, _ = argon2.VerifyEncoded([]byte(utils.ToString(pass)), []byte(utils.ToString(pwd)))
				}
			}
		}
		if valid {
			// when password matching
			token := l.MySession(utils.ToString(log), utils.Compare(response[0]["super_admin"], true), false) // update session variables
			response[0]["token"] = token
			l.Response(response, nil, "", "")
			return
		}
		l.Response(utils.Results{}, errors.New("AUTH : password invalid"), "", "") // API response
		return
	}
	l.Response(utils.Results{}, errors.New("AUTH : can't find login data"), "", "")
}

// @Title Logout
// @Description User logout
// @Param	body		body 	Credential	true		"Credentials"
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @Failure 402 user already connected
// @router /logout [get]
func (l *AuthController) Logout() {
	login, superAdmin, err := l.IsAuthorized() // check if already connected
	if err != nil {
		l.Response(nil, err, "", "")
	}
	l.MySession(login, superAdmin, true) // update session variables
	l.Response(utils.Results{utils.Record{"name": login}}, nil, "", "")
}

// @Title Refresh
// @Description User logout
// @Param	body		body 	Credential	true		"Credentials"
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @Failure 402 user already connected
// @router /logout [get]
func (l *AuthController) Refresh() {
	login, superAdmin, err := l.IsAuthorized() // check if already connected
	if err != nil {
		l.Response(nil, err, "", "")
	}
	token := l.MySession(login, superAdmin, false) // update session variables
	response, err := domain.IsLogged(true, login, token)
	l.Response(response, err, "", "")
}

// @Title Get Maintenance
// @Description Server Maintenance
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @Failure 402 user already connected
// @router /maintenance [get]
func (l *AuthController) GetMaintenance() {
	_, superAdmin, err := l.IsAuthorized() // check if already connected
	if err != nil {
		l.Response(nil, err, "", "")
	}
	if superAdmin {
		l.Response(utils.Results{utils.Record{"is_maintenance": domain.IsMaintenance}}, nil, "", "")
		return
	}
	l.Response(utils.Results{}, errors.New("not allowed to get maintenance mode of the service"), "", "")
}

// @Title Maintenance
// @Description Server Maintenance
// @Param	body		body 	Credential	true		"Credentials"
// @Success 200 {string} success !
// @Failure 403 user does not exist
// @Failure 402 user already connected
// @router /maintenance [post]
func (l *AuthController) Maintenance() {
	_, superAdmin, err := l.IsAuthorized() // check if already connected
	if err != nil {
		l.Response(nil, err, "", "")
	}
	if superAdmin {
		body := l.Body(false) // extracting body
		if log, ok := body["is_maintenance"]; ok {
			domain.IsMaintenance = utils.Compare(log, true)
			l.Response(utils.Results{}, nil, "", "")
			return
		}
	}
	l.Response(utils.Results{}, errors.New("not allowed to change maintenance mode of the service"), "", "")
}

func decrypt(encryptedBase64 string, key []byte, iv []byte) (s string, e error) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	ciphertext, _ := base64.StdEncoding.DecodeString(encryptedBase64)

	block, err := aes.NewCipher(key)
	if err != nil {
		e = err
		return
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// Remove PKCS7 padding
	padding := int(decrypted[len(decrypted)-1])
	s = string(decrypted[:len(decrypted)-padding])
	return
}
