package service

import (
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/specialized_service/email_service"
	favorite "sqldb-ws/domain/specialized_service/favorite_service"
	schema "sqldb-ws/domain/specialized_service/schema_service"
	task "sqldb-ws/domain/specialized_service/task_service"
	user "sqldb-ws/domain/specialized_service/user_service"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"

	"time"
)

// export all specialized services available per domain
var SERVICES = []func() utils.SpecializedServiceITF{
	schema.NewSchemaService,
	schema.NewSchemaFieldsService,
	schema.NewViewService,
	task.NewWorkflowService,
	task.NewTaskService,
	task.NewRequestService,
	favorite.NewFilterService,
	//&favorite.DashboardService{},
	user.NewDelegationService,
	user.NewShareService,
	user.NewUserService,
	email_service.NewEmailResponseService,
	email_service.NewEmailSendedService,
	email_service.NewEmailSendedUserService,
}

// funct to get specialized service depending on table reached
func SpecializedService(name string) utils.SpecializedServiceITF {
	for _, service := range SERVICES {
		if service().Entity().GetName() == name {
			return service()
		}
	}
	return &CustomService{}
}

// Default Specialized Service.
type CustomService struct {
	servutils.SpecializedService
}

func (s *CustomService) Entity() utils.SpecializedServiceInfo { return nil }
func (s *CustomService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	if _, err, ok := servutils.CheckAutoLoad(tablename, record, s.Domain); ok {
		return s.SpecializedService.VerifyDataIntegrity(record, tablename)
	} else {
		return record, err, false
	}

}

// MISSING SOMETHING IMPORTANT: Loop + when delegated if start delayed.
// i need a loop that check every day if delegation is startin or comin to expiry.
// it concerns btw everything containing start_date and end_date.
// we can have simultaneous starting... forkin

func VerifyLoop(domain utils.DomainITF, schemas ...sm.SchemaModel) {
	for _, sch := range schemas {
		currentTime := time.Now()
		if sch.HasField("start_date") && sch.HasField("end_date") {
			sqlFilter := "('" + currentTime.Format("2006-01-02") + "' > end_date)"
			domain.DeleteSuperCall(utils.AllParams(sch.Name), sqlFilter)
			sqlFilter = "'" + currentTime.Format("2006-01-02") + "' > start_date"
			if res, err := domain.SuperCall(utils.AllParams(sch.Name).RootRaw(), utils.Record{}, utils.SELECT, false, sqlFilter); err == nil && len(res) > 0 {
				for _, r := range res {
					SpecializedService(sch.Name).Trigger(r)
				}
			}
		}
	}
}
