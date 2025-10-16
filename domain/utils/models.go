package utils

import (
	"mime/multipart"
	infrastructure "sqldb-ws/infrastructure/service"
)

type AbstractDomain struct {
	Service            infrastructure.InfraServiceItf
	SpecializedService infrastructure.InfraSpecializedServiceItf
	Redirections       []string
	DomainRequestID    string
	TableName          string
	AutoLoad           bool
	User               string
	UserID             string
	Shallowed          bool
	SuperAdmin         bool
	RawView            bool
	Empty              bool
	LowerRes           bool
	Own                bool
	Method             Method
	Params             Params
	File               multipart.File
	FileHandler        *multipart.FileHeader
	SearchInFiles      map[string]string
	IsDraftToPublished bool
	Mode               string
}

func (d *AbstractDomain) GetMode() string {
	return d.Mode
}

func (d *AbstractDomain) GetSpecialized(override string) infrastructure.InfraSpecializedServiceItf {
	return d.SpecializedService
}

func (d *AbstractDomain) AddDetectFileToSearchIn(fileField string, search string) {
	d.SearchInFiles[search] = fileField
}

func (d *AbstractDomain) GetIsDraftToPublished() bool {
	return d.IsDraftToPublished
}

func (d *AbstractDomain) DetectFileToSearchIn() map[string]string {
	return d.SearchInFiles
}

func (d *AbstractDomain) GetUniqueRedirection() string {
	if len(d.Redirections) == 1 {
		return d.Redirections[0]
	}
	return ""
}

func (d *AbstractDomain) GetDomainID() string {
	return d.DomainRequestID
}

func (d *AbstractDomain) SetOwn(own bool) {
	d.Own = own
}
func (d *AbstractDomain) SetAutoload(auto bool) {
	d.AutoLoad = auto
}
func (d *AbstractDomain) GetFile() (multipart.File, *multipart.FileHeader) {
	return d.File, d.FileHandler
}
func (d *AbstractDomain) SetFile(f multipart.File, fh *multipart.FileHeader) {
	d.File = f
	d.FileHandler = fh
}
func (d *AbstractDomain) GetAutoload() bool { return d.AutoLoad }
func (d *AbstractDomain) GetMethod() Method { return d.Method }
func (d *AbstractDomain) GetEmpty() bool    { return d.Empty }
func (d *AbstractDomain) GetUserID() string {
	return d.UserID
}
func (d *AbstractDomain) GetUser() string     { return d.User }
func (d *AbstractDomain) IsSuperAdmin() bool  { return d.SuperAdmin }
func (d *AbstractDomain) IsShallowed() bool   { return d.Shallowed }
func (d *AbstractDomain) GetParams() Params   { return d.Params }
func (d *AbstractDomain) GetTable() string    { return d.TableName }
func (d *AbstractDomain) IsLowerResult() bool { return d.LowerRes }
func (d *AbstractDomain) IsOwn(checkPerm bool, force bool, method Method) bool {
	return d.Own
}
