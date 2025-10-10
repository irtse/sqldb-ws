// compile with: go build -buildmode=plugin -o plugin.so plugin.go

// plugin.go
package main

import (
	"sqldb-ws/domain/domain_service/filter"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	service "sqldb-ws/domain/specialized_service"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	models "sqldb-ws/plugins/datas"
	"strings"
)

func Autoload() []sm.SchemaModel {
	ds.OWNPERMISSIONEXCEPTION = append(ds.OWNPERMISSIONEXCEPTION, []string{
		models.CoCFR.Name, models.ProjectFR.Name, models.Axis.Name,
		models.ProofreadingStatus.Name, models.MajorConference.Name,
		models.PublicationStatusFR.Name, ds.DBUser.Name}...)

	ds.PERMISSIONEXCEPTION = append(ds.PERMISSIONEXCEPTION, []string{
		models.CoCFR.Name, models.ProjectFR.Name, models.Axis.Name,
		models.ProofreadingStatus.Name, models.MajorConference.Name,
		models.PublicationStatusFR.Name,
		models.OtherPublicationAuthorsFR.Name,
		models.OtherPublicationAffiliationAuthorsFR.Name,
		models.ArticleAuthorsFR.Name,
		models.ArticleAffiliationAuthorsFR.Name,
		models.ConferenceAuthorsFR.Name,
		models.ConferenceAffiliationAuthorsFR.Name,
		models.DemoAuthorsFR.Name,
		models.DemoAffiliationAuthorsFR.Name,
		models.HDRAuthorsFR.Name,
		models.HDRAffiliationAuthorsFR.Name,
		models.InternshipAuthorsFR.Name,
		models.InternshipAffiliationAuthorsFR.Name,
		models.PosterAuthorsFR.Name,
		models.PosterAffiliationAuthorsFR.Name,
		models.PresentationAuthorsFR.Name,
		models.PresentationAffiliationAuthorsFR.Name,
		models.ThesisAuthorsFR.Name,
		models.ThesisAffiliationAuthorsFR.Name,
		models.ArticleFR.Name, models.OtherPublicationFR.Name,
		models.DemoFR.Name, models.InternshipFR.Name, models.ThesisFR.Name, models.HDRFR.Name,
		models.PosterFR.Name, models.PresentationFR.Name, models.ConferenceFR.Name,
	}...)
	ds.POSTPERMISSIONEXCEPTION = append(ds.POSTPERMISSIONEXCEPTION, []string{
		models.OtherPublicationAuthorsFR.Name,
		models.OtherPublicationAffiliationAuthorsFR.Name,
		models.ArticleAuthorsFR.Name,
		models.ArticleAffiliationAuthorsFR.Name,
		models.ConferenceAuthorsFR.Name,
		models.ConferenceAffiliationAuthorsFR.Name,
		models.DemoAuthorsFR.Name,
		models.DemoAffiliationAuthorsFR.Name,
		models.HDRAuthorsFR.Name,
		models.HDRAffiliationAuthorsFR.Name,
		models.InternshipAuthorsFR.Name,
		models.InternshipAffiliationAuthorsFR.Name,
		models.PosterAuthorsFR.Name,
		models.PosterAffiliationAuthorsFR.Name,
		models.PresentationAuthorsFR.Name,
		models.PresentationAffiliationAuthorsFR.Name,
		models.ThesisAuthorsFR.Name,
		models.ThesisAffiliationAuthorsFR.Name,

		models.ArticleFR.Name,
		models.OtherPublicationFR.Name,
		models.DemoFR.Name, models.InternshipFR.Name, models.ThesisFR.Name, models.HDRFR.Name,
		models.PosterFR.Name, models.PresentationFR.Name, models.ConferenceFR.Name,
	}...)
	service.SERVICES = append(service.SERVICES, []func() utils.SpecializedServiceITF{NewPublicationService, NewPosterService, NewConferenceService, NewPresentationService}...)
	return []sm.SchemaModel{models.CoCFR, models.ProjectFR, models.Axis, models.MajorConference,
		models.OtherPublicationFR, models.DemoFR, models.InternshipFR, models.ThesisFR, models.HDRFR,
		models.PosterFR, models.PresentationFR, models.ConferenceFR,
		models.PublicationStatusFR, models.ArticleFR,
		models.OtherPublicationAuthorsFR,
		models.OtherPublicationAffiliationAuthorsFR,
		models.ArticleAuthorsFR,
		models.ArticleAffiliationAuthorsFR,
		models.ConferenceAuthorsFR,
		models.ConferenceAffiliationAuthorsFR,
		models.DemoAuthorsFR,
		models.ProofreadingStatus,
		models.DemoAffiliationAuthorsFR,
		models.HDRAuthorsFR,
		models.HDRAffiliationAuthorsFR,
		models.InternshipAuthorsFR,
		models.InternshipAffiliationAuthorsFR,
		models.PosterAuthorsFR,
		models.PosterAffiliationAuthorsFR,
		models.PresentationAuthorsFR,
		models.PresentationAffiliationAuthorsFR,
		models.ThesisAuthorsFR,
		models.ThesisAffiliationAuthorsFR,
	}
}

// article, conference, présentation, thèse, stage, démo, autre, HDR, poster
// DONE - ~ 200 LINES - PARTIALLY TESTED
type PublicationService struct {
	servutils.AbstractSpecializedService
}

func NewPublicationService() utils.SpecializedServiceITF {
	return &PublicationService{AbstractSpecializedService: servutils.AbstractSpecializedService{
		ManyToMany: map[string][]map[string]interface{}{},
		OneToMany:  map[string][]map[string]interface{}{},
	},
	}
}

func (s *PublicationService) Entity() utils.SpecializedServiceInfo { return models.OtherPublicationFR }

func (s *PublicationService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	ok := record["major_conference"]
	isNotFound := true
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.MajorConference.Name, map[string]interface{}{}, false); err == nil && len(res) > 0 {
		for _, r := range res {
			if strings.Contains(strings.ToUpper(utils.GetString(record, "major_conference_name")), strings.ToUpper(utils.GetString(r, "name"))) {
				ok = "yes"
				isNotFound = false
				break
			}
		}
	}
	if isNotFound {
		ok = "no"
	}
	record["major_conference"] = ok
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *PublicationService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), false, innerestr...)
}

type PosterService struct {
	servutils.AbstractSpecializedService
}

func NewPosterService() utils.SpecializedServiceITF {
	return &PosterService{AbstractSpecializedService: servutils.AbstractSpecializedService{
		ManyToMany: map[string][]map[string]interface{}{},
		OneToMany:  map[string][]map[string]interface{}{},
	},
	}
}

func (s *PosterService) Entity() utils.SpecializedServiceInfo { return models.PosterFR }

func (s *PosterService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	ok := record["major_conference"]
	isNotFound := true
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.MajorConference.Name, map[string]interface{}{}, false); err == nil && len(res) > 0 {
		for _, r := range res {
			if strings.Contains(strings.ToUpper(utils.GetString(record, "conference_name")), strings.ToUpper(utils.GetString(r, "name"))) {
				ok = "yes"
				isNotFound = false
				break
			}
		}
	}
	if isNotFound {
		ok = "no"
	}
	record["major_conference"] = ok
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *PosterService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), false, innerestr...)
}

type PresentationService struct {
	servutils.AbstractSpecializedService
}

func NewPresentationService() utils.SpecializedServiceITF {
	return &PresentationService{AbstractSpecializedService: servutils.AbstractSpecializedService{
		ManyToMany: map[string][]map[string]interface{}{},
		OneToMany:  map[string][]map[string]interface{}{},
	},
	}
}

func (s *PresentationService) Entity() utils.SpecializedServiceInfo { return models.PresentationFR }

func (s *PresentationService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	ok := record["major_conference"]
	isNotFound := true
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.MajorConference.Name, map[string]interface{}{}, false); err == nil && len(res) > 0 {
		for _, r := range res {
			if strings.Contains(strings.ToUpper(utils.GetString(record, "conference_name")), strings.ToUpper(utils.GetString(r, "name"))) {
				ok = "yes"
				isNotFound = false
				break
			}
		}
	}
	if isNotFound {
		ok = "no"
	}
	record["major_conference"] = ok
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *PresentationService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), false, innerestr...)
}

type ConferenceService struct {
	servutils.AbstractSpecializedService
}

func NewConferenceService() utils.SpecializedServiceITF {
	return &ConferenceService{AbstractSpecializedService: servutils.AbstractSpecializedService{
		ManyToMany: map[string][]map[string]interface{}{},
		OneToMany:  map[string][]map[string]interface{}{},
	},
	}
}

func (s *ConferenceService) Entity() utils.SpecializedServiceInfo { return models.ConferenceFR }

func (s *ConferenceService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	ok := record["major_conference"]
	isNotFound := true
	if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.MajorConference.Name, map[string]interface{}{}, false); err == nil && len(res) > 0 {
		for _, r := range res {
			if strings.Contains(strings.ToUpper(utils.GetString(record, "major_conference_name")), strings.ToUpper(utils.GetString(r, "name"))) {
				ok = "yes"
				isNotFound = false
				break
			}
		}
	}
	if isNotFound {
		ok = "no"
	}
	record["major_conference"] = ok
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *ConferenceService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), false, innerestr...)
}
