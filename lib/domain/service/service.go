package service

import ( 
	"fmt"
	tool "sqldb-ws/lib" 
	"sqldb-ws/lib/entities" 
	conn "sqldb-ws/lib/infrastructure/connector" 
	task "sqldb-ws/lib/domain/service/task_service" 
	user "sqldb-ws/lib/domain/service/user_service"
	schema "sqldb-ws/lib/domain/service/schema_service" 
)

// export all specialized services available per domain
var SERVICES = []tool.SpecializedService{
	&schema.SchemaService{}, 
	&schema.SchemaFields{}, 
	&schema.ViewService{},
	&schema.ActionService{},
	&task.TaskAssigneeService{}, 
	&task.TaskVerifyerService{}, 
	&task.TaskService{},
	&task.TaskWatcherService{},
	&user.UserEntityService{},
	&user.HierarchyService{},
	&user.RoleAttributionService{},
	&user.RoleService{},
	&user.EntityService{},
	&user.PermissionService{},
	&user.UserService{},
}
// funct to get specialized service depending on table reached
func SpecializedService(name string) tool.SpecializedService {
	for _, service := range SERVICES {
		if service.Entity().GetName() == name { return service }
	}
	return &CustomService{}
}
// Default Specialized Service. 
type CustomService struct { tool.AbstractSpecializedService }
func (s *CustomService) UpdateRowAutomation(results tool.Results, record tool.Record) {

}
func (s *CustomService) WriteRowAutomation(record tool.Record, tableName string) {
	s.Domain.WriteRow(tableName, record) // call main func that link creator user and new row
}
func (s *CustomService) DeleteRowAutomation(results tool.Results, tableName string) {
	s.Domain.DeleteRow(tableName, results) // call main func that unlink creator user and new row
}
func (s *CustomService) Entity() tool.SpecializedServiceInfo { return nil }
func (s *CustomService) VerifyRowAutomation(record tool.Record, create bool) (tool.Record, bool, bool) { 
	return record, true, false }
func (s *CustomService) PostTreatment(results tool.Results, tableName string, dest_id... string) tool.Results { 
	return s.Domain.PostTreat( results, tableName, false) // call main post treatment
}
// default have a right to access to whatever is in dbuser_entry database... (sets at creation)
func (s *CustomService) ConfigureFilter(tableName string) (string, string) {
	restr := "id IN (SELECT " + fmt.Sprintf("%v",  entities.RootID("dest_table")) + " FROM " + entities.DBUserEntry.Name 
	restr += " WHERE " + fmt.Sprintf("%v", entities.RootID(entities.DBSchema.Name))  + " IN ("
	restr += "SELECT id FROM " + entities.DBSchema.Name + " WHERE name=" + tableName + ") "
	restr += "AND " + fmt.Sprintf("%v",  entities.RootID(entities.DBUser.Name)) + " IN (SELECT id FROM " + entities.DBUser.Name 
	restr += " WHERE login=" + conn.Quote(s.Domain.GetUser()) + "))"
	return s.Domain.ViewDefinition(tableName, restr)
}	
// to set up ConfigureFilter