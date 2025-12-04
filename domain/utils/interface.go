package utils

import (
	conn "sqldb-ws/infrastructure/connector/db"
	infrastructure "sqldb-ws/infrastructure/service"
)

type SpecializedServiceITF interface {
	SetDomain(d DomainITF) SpecializedServiceITF
	Entity() SpecializedServiceInfo
	TransformToGenericView(results Results, tableName string, dest_id ...string) Results
	Trigger(record map[string]interface{}, db *conn.Database)
	infrastructure.InfraSpecializedServiceItf
}
type SpecializedServiceInfo interface{ GetName() string }

type DomainITF interface {
	// Main Procedure of services at Domain level.
	GetIsDraftToPublished() bool
	GetRecord() Record
	SetTrace(trace bool)
	GetSpecialized(override string) infrastructure.InfraSpecializedServiceItf
	AddDetectFileToSearchIn(fileField string, search string)
	DetectFileToSearchIn() map[string]string
	SuperCall(params Params, record Record, method Method, isOwn bool, args ...interface{}) (Results, error)
	CreateSuperCall(params Params, record Record, trace bool, args ...interface{}) (Results, error)
	UpdateSuperCall(params Params, rec Record, trace bool, args ...interface{}) (Results, error)
	DeleteSuperCall(params Params, trace bool, args ...interface{}) (Results, error)
	Call(params Params, rec Record, m Method, args ...interface{}) (Results, error)

	// Main accessor defined by DomainITF interface
	GetUniqueRedirection() string
	GetDomainID() string
	GetMode() string
	GetDb() *conn.Database
	SetDb(db *conn.Database)
	GetMethod() Method
	GetTable() string
	GetUserID() string
	GetUser() string
	GetEmpty() bool
	GetParams() Params
	GetAutoload() bool
	SetAutoload(auto bool)
	// Main accessor defined by DomainITF interface
	HandleRecordAttributes(record Record)

	// Main accessor defined by DomainITF interface
	SetOwn(own bool)
	IsOwn(checkPerm bool, force bool, method Method) bool
	IsSuperAdmin() bool
	IsShallowed() bool
	IsLowerResult() bool

	VerifyAuth(tableName string, colName string, level string, method Method, dest ...string) bool
}
