package schema

import (
	"fmt"
	"os"
	"plugin"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

func Load(domainInstance utils.DomainITF) {
	db := connector.Open(nil)
	defer db.Close()
	progressbar.OptionSetMaxDetailRow(1)
	demoTable := []sm.SchemaModel{}
	if os.Getenv("PLUGINS") != "" {
		for _, plug := range strings.Split(os.Getenv("PLUGINS"), ",") {
			if p, err := plugin.Open("./plugins/autoload_" + plug + "/plugin.so"); err == nil {
				if sym, err := p.Lookup("Autoload"); err == nil {
					launchFunc := sym.(func() []sm.SchemaModel)
					demoTable = append(demoTable, launchFunc()...)
				}
			} else {
				fmt.Println(err)
			}
		}
	}
	bar := progressbar.NewOptions64(
		int64(len(ds.NOAUTOLOADROOTTABLES)+len(ds.ROOTTABLES)+len(demoTable)+len(ds.DBRootViews)+1),
		progressbar.OptionSetDescription("Setup root DB"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(10),
		progressbar.OptionShowTotalBytes(true),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
		progressbar.OptionSetMaxDetailRow(1),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
	domainInstance.SetAutoload(true)
	LoadCache(utils.ReservedParam, db)
	InitializeTables(domainInstance, bar)                // Create tables if they don't exist, needed for the next step
	InitializeRootTables(domainInstance, demoTable, bar) // Create root tables if they don't exist, needed for the next step
	CreateSuperAdmin(domainInstance, bar)
	CreateRootView(domainInstance, bar)
}

func InitializeTables(domainInstance utils.DomainITF, bar *progressbar.ProgressBar) {
	for _, table := range ds.NOAUTOLOADROOTTABLES {
		bar.AddDetail("Creating table " + table.Name)
		domainInstance.CreateSuperCall(utils.GetTableTargetParameters(table.Name).RootRaw(), table.ToSchemaRecord())
		bar.Add(1)
	}
}

func InitializeRootTables(domainInstance utils.DomainITF, demoTable []sm.SchemaModel, bar *progressbar.ProgressBar) {
	var wfNew, viewNew bool
	rootTables := append(ds.ROOTTABLES, demoTable...)
	for _, table := range rootTables {
		if _, err := GetSchema(table.Name); err != nil {
			bar.AddDetail("Creating Schema " + table.Name)
			r := table.ToRecord()
			for _, field := range table.Fields {
				for _, f := range utils.ToList(r["fields"]) {
					if utils.ToString(utils.ToMap(f)["name"]) == field.Name {
						utils.ToMap(f)["foreign_table"] = field.ForeignTable
					}
				}
				if schema, err := GetSchema(field.ForeignTable); err == nil {
					for _, f := range utils.ToList(r["fields"]) {
						if utils.ToString(utils.ToMap(f)["name"]) == field.Name {
							utils.ToMap(f)["link_id"] = schema.ID
						}
					}
				}
			}
			if CreateRootTable(domainInstance, r) {
				wfNew = table.Name == ds.DBWorkflow.Name
				viewNew = table.Name == ds.DBView.Name
				if schema, err := GetSchema(table.Name); err == nil {
					if wfNew {
						CreateWorkflowView(domainInstance, schema, bar)
					}
					if viewNew {
						CreateView(domainInstance, schema, bar)
					}
				}
			}
		}
		bar.Add(1)
	}
}

func CreateRootTable(domainInstance utils.DomainITF, record utils.Record) bool {
	params := utils.AllParams(ds.DBSchema.Name).RootRaw()
	res, err := domainInstance.CreateSuperCall(params, record)
	return !(err != nil || len(res) == 0)
}

func CreateWorkflowView(domainInstance utils.DomainITF, schema sm.SchemaModel, bar *progressbar.ProgressBar) {
	bar.AddDetail("Creating Integration Workflow for Schema " + schema.Name)
	params := utils.Params{
		Values: map[string]string{
			utils.RootTableParam: ds.DBView.Name,
			utils.RootRowsParam:  utils.ReservedParam,
			utils.RootRawView:    "enable",
		},
		Mutex: &sync.RWMutex{},
	}
	newWorkflow := utils.Record{
		sm.NAMEKEY:       "workflow",
		"indexable":      true,
		"description":    fmt.Sprintf("View description for %s datas.", ds.DBWorkflow.Name),
		"category":       "workflow",
		"is_empty":       false,
		"index":          0,
		"is_list":        true,
		"readonly":       false,
		ds.SchemaDBField: schema.ID,
	}
	domainInstance.CreateSuperCall(params.RootRaw(), newWorkflow)
}

func CreateRootView(domainInstance utils.DomainITF, bar *progressbar.ProgressBar) {
	for _, view := range ds.DBRootViews {
		bar.AddDetail("Creating Root View " + utils.ToString(utils.ToMap(view)[sm.NAMEKEY]))
		params := utils.AllParams(ds.DBView.Name).RootRaw()
		r, err := domainInstance.CreateSuperCall(params, view)
		if err != nil || len(r) == 0 {
			bar.Add(1)
			continue
		}
		realView := r[0]
		sch, err := GetSchema(utils.ToString(view["foreign_table"]))
		if err == nil {
			realView[ds.SchemaDBField] = sch.ID
			delete(view, "foreign_table")
			if filter, ok := view["filter"]; ok {
				delete(view, "filter")
				mFilter := utils.ToMap(filter)
				attr := "fields"
				if _, ok := mFilter["view_fields"]; ok {
					attr = "view_fields"
				}
				if res, err := domainInstance.CreateSuperCall(utils.AllParams(ds.DBFilter.Name).RootRaw(), utils.Record{
					sm.NAMEKEY:       utils.ToString(mFilter[sm.NAMEKEY]),
					"is_view":        attr == "view_fields",
					ds.SchemaDBField: sch.ID,
				}); err == nil && len(res) > 0 {
					if fields := mFilter[attr]; fields != nil {
						f := utils.Record{ds.FilterDBField: res[0][utils.SpecialIDParam]}
						for _, field := range utils.ToList(fields) {
							if n, ok := utils.ToMap(field)["name"]; ok {
								if ff, err := sch.GetField(utils.ToString(n)); err == nil {
									utils.ToMap(field)[ds.SchemaFieldDBField] = ff.ID
								}
								delete(utils.ToMap(field), "name")
							}
							for k, v := range utils.ToMap(field) {
								f[k] = v
							}
						}
						domainInstance.CreateSuperCall(utils.AllParams(ds.DBFilterField.Name).RootRaw(), f)
					}
					realView[ds.FilterDBField] = res[0][utils.SpecialIDParam]
				}
			}
		}
		delete(realView, "name")
		domainInstance.UpdateSuperCall(utils.AllParams(ds.DBView.Name).Enrich(map[string]interface{}{
			"id": realView["id"],
		}).RootRaw(), realView)
		bar.Add(1)
	}
}

func CreateView(domainInstance utils.DomainITF, schema sm.SchemaModel, bar *progressbar.ProgressBar) {
	bar.AddDetail("Create View for Schema " + schema.Name)
	params := utils.AllParams(ds.DBWorkflow.Name).RootRaw()
	newView := utils.Record{
		sm.NAMEKEY:       fmt.Sprintf("create %s", ds.DBView.Name),
		"description":    fmt.Sprintf("new %s workflow", ds.DBView.Name),
		ds.SchemaDBField: schema.ID,
	}
	domainInstance.CreateSuperCall(params, newView)
}

func CreateSuperAdmin(domainInstance utils.DomainITF, bar *progressbar.ProgressBar) {
	bar.AddDetail("Create SuperAdmin profile user ")
	domainInstance.CreateSuperCall(utils.AllParams(ds.DBUser.Name).RootRaw(), utils.Record{
		"name":        os.Getenv("SUPERADMIN_NAME"),
		"email":       os.Getenv("SUPERADMIN_EMAIL"),
		"super_admin": true,
		"password":    os.Getenv("SUPERADMIN_PASSWORD"),
	})
	bar.AddDetail("")
	bar.Add(1)
}
