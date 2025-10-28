package controller

import (
	"fmt"
	"os"

	"github.com/go-ldap/ldap/v3"
)

// ConnectToLDAPServer se connecte au serveur LDAP avec les identifiants donnés
func ConnectToLDAPServer(ldapServer string, username string, password string) (*ldap.Conn, error) {
	l, err := ldap.DialURL(ldapServer)
	if err != nil {
		return nil, err
	}

	err = l.Bind(username, password)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// ListAllUsers liste tous les utilisateurs dans l'annuaire
func ListAllUsers(l *ldap.Conn, baseDN string, pagingSize uint32) ([]string, error) {
	var users []string

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=person)",
		[]string{"dn"},
		nil,
	)

	pagingControl := ldap.NewControlPaging(pagingSize)

	for {
		searchRequest.Controls = []ldap.Control{pagingControl}

		result, err := l.Search(searchRequest)
		if err != nil {
			return nil, err
		}

		for _, entry := range result.Entries {
			users = append(users, entry.DN)
		}

		updatedControl := ldap.FindControl(result.Controls, ldap.ControlTypePaging)
		if ctrl, ok := updatedControl.(*ldap.ControlPaging); ok {
			if len(ctrl.Cookie) == 0 {
				break
			} else {
				pagingControl.SetCookie(ctrl.Cookie)
			}
		} else {
			break
		}
	}

	return users, nil
}

// GetUserDN récupère le DN d'un utilisateur à partir de son sAMAccountName
func GetUserDN(l *ldap.Conn, baseDN string, sAMAccountName string) (string, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", sAMAccountName),
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return "", err
	}

	if len(sr.Entries) != 1 {
		return "", fmt.Errorf("User does not exist or too many entries returned")
	}

	return sr.Entries[0].DN, nil
}

// CheckUserCredentials vérifie les identifiants d'un utilisateur
func CheckUserCredentials(l *ldap.Conn, baseDN string, sAMAccountName string, password string) bool {
	userDN, err := GetUserDN(l, baseDN, sAMAccountName)
	if err != nil {
		return false
	}

	err = l.Bind(userDN, password)
	return err == nil
}

// GetUserInfos récupère les informations d'un utilisateur à partir de son sAMAccountName
func GetUserInfos(l *ldap.Conn, baseDN string, sAMAccountName string) (map[string][]string, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", sAMAccountName),
		nil, // Ne spécifiez aucun attribut pour obtenir tous les attributs disponibles
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(sr.Entries) != 1 {
		return nil, fmt.Errorf("User does not exist or too many entries returned")
	}

	userInfos := make(map[string][]string)
	for _, attr := range sr.Entries[0].Attributes {
		userInfos[attr.Name] = attr.Values
	}

	return userInfos, nil
}

func CheckLdap(user string, password string) bool {
	// Exemple d'utilisation
	address := os.Getenv("LDAP_ADDR")
	if address == "" {
		address = "ldap://irt00v001.irt-aese.local:389"
	}
	username := os.Getenv("LDAP_USR")
	if username == "" {
		username = "CN=read only,OU=Utilisateurs,OU=YZ - Technique,OU=AD - IRT-AESE,DC=IRT-AESE,DC=local"
	}
	ou := os.Getenv("LDAP_OU")
	if ou == "" {
		ou = "OU=AD - IRT-AESE,DC=IRT-AESE,DC=local"
	}
	pwd := os.Getenv("LDAP_PWD")
	if pwd == "" {
		pwd = "IRT@2015"
	}
	conn, err := ConnectToLDAPServer(address, username, pwd)
	if err != nil {
		fmt.Println("LDAP ERR : ", err)
		return false
	}
	defer conn.Close()

	if users, err := ListAllUsers(conn, ou, 500); err != nil || len(users) == 0 {
		fmt.Println("LDAP2 ERR : ", err)
		return false
	}
	return CheckUserCredentials(conn, ou, user, password)
}
