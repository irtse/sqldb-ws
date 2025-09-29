package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"plugin"
	domain "sqldb-ws/domain"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	specialized "sqldb-ws/domain/specialized_service"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	_ "sqldb-ws/routers"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/matthewhartstonge/argon2"
)

func main() {
	title := "  _____  ____  _      _____  ____     __          _______ \n"
	title += " / ____|/ __ \\| |    |  __ \\|  _ \\    \\ \\        / / ____|\n"
	title += "| (___ | |  | | |    | |  | | |_) |____\\ \\  /\\  / / (___  \n"
	title += " \\___ \\| |  | | |    | |  | |  _ <______\\ \\/  \\/ / \\___ \\ \n"
	title += " ____) | |__| | |____| |__| | |_) |      \\  /\\  /  ____) |\n"
	title += "|_____/ \\___\\_\\______|_____/|____/        \\/  \\/  |_____/ \n"
	title += "														 "
	fmt.Printf("%s\n", title)
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}
	beego.SetStaticPath("/", "web")
	for key, value := range DEFAULTCONF {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	if os.Getenv("SUPERADMIN_PASSWORD") != "" {
		argon := argon2.DefaultConfig()
		hash, _ := argon.HashEncoded([]byte(os.Getenv("SUPERADMIN_PASSWORD")))
		os.Setenv("SUPERADMIN_PASSWORD", string(hash))
	}

	fmt.Printf("%s\n", "Service in "+os.Getenv("AUTH_MODE")+" mode")
	schema.Load(domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil))
	fmt.Printf("%s\n", "Running server...")
	if os.Getenv("PLUGINS") != "" {
		for _, plug := range strings.Split(os.Getenv("PLUGINS"), ",") {
			if p, err := plugin.Open("./plugins/" + plug + "/plugin.so"); err == nil {
				if sym, err := p.Lookup("Run"); err == nil {
					launchFunc := sym.(func())
					go launchFunc()
				}
			} else {
				fmt.Println(err)
			}
		}
	}
	VerifyData()
	GetResponse()
	beego.Run()
}

func VerifyData() {
	for true {
		go specialized.VerifyLoop(domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil))
		time.Sleep(24 * time.Hour)
	}
}

func GetResponse() {
	url := os.Getenv("RESPONSE_URL")
	if url == "" {
		fmt.Println("No response URL to reach...")
		return
	}
	for true {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		defer resp.Body.Close() // always close the response body

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		var b map[string]interface{}
		json.Unmarshal(body, &b)
		datas := b["data"].(map[string]interface{})
		go func() {
			for code, data := range datas {
				d := domain.Domain(true, "", nil)
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEmailSended.Name, map[string]interface{}{
					"code": connector.Quote(code),
				}, false); err == nil && len(res) > 0 {
					emailRelated := res[0]
					d.CreateSuperCall(utils.AllParams(ds.DBEmailResponse.Name).Enrich(map[string]interface{}{
						"code": code,
					}).RootRaw(), map[string]interface{}{
						"got_response":        data,
						ds.EmailSendedDBField: emailRelated[utils.SpecialIDParam],
					})
				}
			}
		}()
		time.Sleep(1 * time.Hour)
	}
}

var DEFAULTCONF = map[string]string{
	"SUPERADMIN_NAME":     "root",
	"SUPERADMIN_PASSWORD": "admin",
	"PLUGINS":             "cegid",
	"SUPERADMIN_EMAIL":    "pro.morgane.roques@gmail.com",
	"AUTH_MODE":           "ldap",
	"DBDRIVER":            "postgres",
	"DBHOST":              "127.0.0.1",
	"DBPORT":              "5432",
	"DBUSER":              "test",
	"DBPWD":               "test",
	"DBNAME":              "test",
	"DBSSL":               "disable",
	"log":                 "disable",
}

// irt-aese.local
