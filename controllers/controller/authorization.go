package controller

import (
	"errors"
	"fmt"
	"os"
	"sqldb-ws/domain"
	"sqldb-ws/domain/utils"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// session is the main manager from Handler
func (t *AbstractController) MySession(userId string, superAdmin bool, delete bool) string {
	var err error
	token := ""
	delFunc := func() { // set up a lambda call back function to delete in session and token in base if needed
		if os.Getenv("AUTH_MODE") != AUTHMODE[0] { // in case of token way of authenticate
			domain.SetToken(superAdmin, userId, nil)
		} else {
			t.DelSession(SESSIONS_KEY) // user_id key
			t.DelSession(ADMIN_KEY)    // super_admin key
		}
	}
	if delete {
		delFunc()
		return token
	} // if only deletion quit after launching lambda

	if os.Getenv("AUTH_MODE") != AUTHMODE[0] { // if token way of authentication
		tokenService := &Token{} // generate a new token with all needed claims
		token, err = tokenService.Create(userId, superAdmin)
		if err != nil {
			t.Response(utils.Results{}, err, "", "")
			return token
		} // then update user with its brand new token.
		domain.SetToken(superAdmin, userId, token)
	} else {
		t.SetSession(SESSIONS_KEY, userId) // load superadmin and user id in session in any case
		t.SetSession(ADMIN_KEY, superAdmin)
	}
	// launch a 24h session timer after this session will be killed.
	go func() {
		timer := time.AfterFunc(time.Hour*24, delFunc)
		defer timer.Stop()
	}()

	return token
}

// authorized is authentication check up func of the HANDLER
func (t *AbstractController) IsAuthorized() (string, bool, error) {
	found := false
	for _, mode := range AUTHMODE { // above all check for kind of auth (token in authorization header, or session API)
		if mode == os.Getenv("AUTH_MODE") {
			found = true
		}
	} // if none found give an error
	if !found {
		return "", false, errors.New("authmode not allowed <" + os.Getenv("AUTH_MODE") + ">")
	}
	// session auth will look in session variables in API ONLY
	if os.Getenv("AUTH_MODE") == AUTHMODE[0] {
		if t.GetSession(SESSIONS_KEY) != nil {
			return utils.ToString(t.GetSession(SESSIONS_KEY)), utils.Compare(t.GetSession(ADMIN_KEY), true), nil
		}
		return "", false, errors.New("user not found in session")
	} // TOKEN verification is a little bit verbose by extractin token in Authorization Header and look after its properties
	header := t.Ctx.Request.Header
	a, ok := header["Authorization"] // extract token in HEADER
	if !ok {
		return "", false, errors.New("no authorization in header")
	}
	tokenService := &Token{}
	token, err := tokenService.Verify(a[0]) // Verify if token is valid
	if err != nil {
		return "", false, err
	}
	claims := token.Claims.(jwt.MapClaims)
	if user_id, ok := claims[SESSIONS_KEY]; ok { // if all in claims send back super mode and user as confirmation
		return utils.ToString(user_id), utils.Compare(claims[ADMIN_KEY], true), nil
	}
	return "", false, errors.New("user not found in token")
}

var SESSIONS_KEY = "user_id"
var ADMIN_KEY = "super_admin"
var AUTHMODE = []string{"session", "token", "ldap"}

var secret = []byte("weakest-secret")

type Token struct{}

func (t *Token) Create(user_id string, superAdmin bool) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		SESSIONS_KEY: user_id,
		ADMIN_KEY:    superAdmin,
		"exp":        time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

func (t *Token) Verify(tokenStr string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	tokenStr, err = token.SignedString(secret)
	return token, err
}
