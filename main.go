package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"plugin"
	domain "sqldb-ws/domain"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
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
				fmt.Println("PLUGIN ERROR :", err)
			}
		}
	}
	go VerifyData()
	go GetResponse()
	beego.Run()
}

func VerifyData() {
	registries := []sm.SchemaModel{}
	for _, sch := range sm.SchemaRegistry {
		registries = append(registries, sch)
	}
	d := domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil)
	for true {
		go specialized.VerifyLoop(d, registries...)
		time.Sleep(24 * time.Hour)
	}
}

func GetResponse() {
	host := os.Getenv("HOST")
	if host == "" {
		fmt.Println("No response URL to reach...")
		return
	}
	for {
		fmt.Println("Retrieve Response")
		resp, err := http.Get(fmt.Sprintf("%s/v1/response", host))
		if err != nil {
			fmt.Println("GetResponse Error:", err)
			time.Sleep(10 * time.Minute)
			continue
		}
		defer resp.Body.Close() // always close the response body

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("GetResponse", "Error:", err)
			time.Sleep(10 * time.Minute)
			continue
		}
		var b map[string]interface{}
		json.Unmarshal(body, &b)
		if b["error"] != nil {
			fmt.Println("GetResponse", b["error"])
			time.Sleep(10 * time.Minute)
			continue
		}
		if b["data"] == nil {
			fmt.Println("GetResponse", "Empty !")
			time.Sleep(10 * time.Minute)
			continue
		}
		datas := b["data"].(map[string]interface{})
		fmt.Println("KEYZ", datas)
		go func() {
			m := map[string]map[string]interface{}{}
			for code, data := range datas {
				if m[strings.ReplaceAll(code, "_str", "")] == nil {
					m[strings.ReplaceAll(code, "_str", "")] = map[string]interface{}{}
				}
				if strings.Contains(code, "_str") {
					decoded, err := url.QueryUnescape(fmt.Sprintf("%v", data))
					if err == nil {
						m[strings.ReplaceAll(code, "_str", "")]["comment"] = strings.ReplaceAll(decoded, "''", "'")
					}
				} else {
					m[strings.ReplaceAll(code, "_str", "")]["got_response"] = data
				}
			}
			d := domain.Domain(true, "", nil)
			for code, data := range m {
				if res, err := d.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBEmailSended.Name, map[string]interface{}{
					"code": connector.Quote(code),
				}, false); err == nil && len(res) > 0 {
					emailRelated := res[0]
					data[ds.EmailSendedDBField] = emailRelated[utils.SpecialIDParam]
					data["update_date"] = time.Now().UTC()
					_, err := d.CreateSuperCall(utils.AllParams(ds.DBEmailResponse.Name).Enrich(map[string]interface{}{
						"code": code,
					}).RootRaw(), data)
					fmt.Println("Is Created ?", err)
				}
			}
		}()
		time.Sleep(10 * time.Minute)
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
