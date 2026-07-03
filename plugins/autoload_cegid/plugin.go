// compile with: go build -buildmode=plugin -o plugin.so plugin.go

// plugin.go
package main

import (
	"fmt"
	"os"
	"sqldb-ws/domain"
	"sqldb-ws/domain/domain_service/filter"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	service "sqldb-ws/domain/specialized_service"
	servutils "sqldb-ws/domain/specialized_service/utils"
	"sqldb-ws/domain/utils"
	models "sqldb-ws/plugins/datas"
	"strings"
)

func Autoload() []sm.SchemaModel {
	s := domain.Domain(true, os.Getenv("SUPERADMIN_NAME"), nil)
	for _, sch := range []string{models.PresentationFR.Name, models.ConferenceFR.Name, models.PosterFR.Name, models.OtherPublicationFR.Name} {
		if resources, err := s.GetDb().ClearQueryFilter().SelectQueryWithRestriction(sch, map[string]interface{}{}, false); err == nil {
			for _, r := range resources {
				ok := false
				isNotFound := true
				if res, err := s.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.MajorConference.Name, map[string]interface{}{}, false); err == nil && len(res) > 0 {
					for _, r := range res {
						if strings.Contains(strings.ToUpper(utils.GetString(r, "conference_acronym")), strings.ToUpper(utils.GetString(r, "name"))) {
							ok = true
							isNotFound = false
							break
						}
					}
				}
				if isNotFound {
					ok = false
				}
				r["major_conference"] = ok
				if ok == true {
					r["reread"] = 1
				}
				err := s.GetDb().ClearQueryFilter().UpdateQuery(sch, r, map[string]interface{}{
					utils.SpecialIDParam: r[utils.SpecialIDParam],
				}, false)
				fmt.Println("ERR", r, err)
			}
		}
	}

	ds.OWNPERMISSIONEXCEPTION = append(ds.OWNPERMISSIONEXCEPTION, []string{
		models.CoCFR.Name, models.ProjectFR.Name, models.Axis.Name,
		models.ProofreadingStatus.Name, models.MajorConference.Name,
		models.PublicationStatusFR.Name, models.PublicationHistoryStatusFR.Name, ds.DBUser.Name}...)

	ds.PERMISSIONEXCEPTION = append(ds.PERMISSIONEXCEPTION, []string{
		models.PublicationTagsFR.Name,
		models.ArticlePublicationTagsFR.Name,
		models.ConferencePublicationTagsFR.Name,
		models.PresentationPublicationTagsFR.Name,
		models.OtherPublicationTagsFR.Name,
		models.DemoPublicationTagsFR.Name,
		models.InternshipPublicationTagsFR.Name,
		models.PosterPublicationTagsFR.Name,
		models.HDRPublicationTagsFR.Name,
		models.ThesisPublicationTagsFR.Name,

		models.CoCFR.Name, models.ProjectFR.Name, models.Axis.Name,
		models.ProofreadingStatus.Name, models.MajorConference.Name,
		models.PublicationStatusFR.Name,
		models.PublicationHistoryStatusFR.Name,
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
		models.ThesisSupervisorAuthorsFR.Name,
		models.ArticleFR.Name, models.OtherPublicationFR.Name,
		models.DemoFR.Name, models.InternshipFR.Name, models.ThesisFR.Name, models.HDRFR.Name,
		models.PosterFR.Name, models.PresentationFR.Name, models.ConferenceFR.Name,
	}...)
	ds.POSTPERMISSIONEXCEPTION = append(ds.POSTPERMISSIONEXCEPTION, []string{
		models.PublicationTagsFR.Name,
		models.ArticlePublicationTagsFR.Name,
		models.ConferencePublicationTagsFR.Name,
		models.PresentationPublicationTagsFR.Name,
		models.OtherPublicationTagsFR.Name,
		models.DemoPublicationTagsFR.Name,
		models.InternshipPublicationTagsFR.Name,
		models.PosterPublicationTagsFR.Name,
		models.HDRPublicationTagsFR.Name,
		models.ThesisPublicationTagsFR.Name,

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
		models.ThesisSupervisorAuthorsFR.Name,
		models.ThesisAffiliationAuthorsFR.Name,

		models.ArticleFR.Name,
		models.OtherPublicationFR.Name,
		models.DemoFR.Name, models.InternshipFR.Name, models.ThesisFR.Name, models.HDRFR.Name,
		models.PosterFR.Name, models.PresentationFR.Name, models.ConferenceFR.Name,
	}...)
	ds.AVOIDUSERPERMISSIONEXCEPTION = append(ds.AVOIDUSERPERMISSIONEXCEPTION, []string{
		models.CoCFR.Name, models.ProjectFR.Name, models.Axis.Name,
		models.ProofreadingStatus.Name, models.MajorConference.Name,
		models.PublicationStatusFR.Name, models.PublicationHistoryStatusFR.Name,
	}...)
	service.SERVICES = append(service.SERVICES, []func() utils.SpecializedServiceITF{
		NewPublicationService(models.OtherPublicationFR),
		NewPublicationService(models.ArticleFR),
		NewPublicationService(models.InternshipFR),
		NewPublicationService(models.ThesisFR),
		NewPublicationService(models.ConferenceFR),
		NewPublicationService(models.PresentationFR),
		NewPublicationService(models.DemoFR),
		NewPublicationService(models.PosterFR),
		NewPublicationService(models.HDRFR),
	}...)
	return []sm.SchemaModel{models.CoCFR, models.ProjectFR, models.Axis, models.MajorConference, models.PublicationTagsFR,
		models.ArticlePublicationTagsFR,
		models.ConferencePublicationTagsFR,
		models.PresentationPublicationTagsFR,
		models.OtherPublicationTagsFR,
		models.DemoPublicationTagsFR,
		models.InternshipPublicationTagsFR,
		models.PosterPublicationTagsFR,
		models.HDRPublicationTagsFR,
		models.ThesisPublicationTagsFR,

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
		models.ThesisSupervisorAuthorsFR,
		models.ThesisAffiliationAuthorsFR,
		models.PublicationHistoryStatusFR,
	}
}

// article, conference, présentation, thèse, stage, démo, autre, HDR, poster
// DONE - ~ 200 LINES - PARTIALLY TESTED
type PublicationService struct {
	servutils.AbstractSpecializedService
	Sch sm.SchemaModel
}

func NewPublicationService(schemaName sm.SchemaModel) func() utils.SpecializedServiceITF {
	if sch, err := schema.GetSchema(schemaName.Name); err == nil {
		schemaName = sch
	}
	return func() utils.SpecializedServiceITF {
		return &PublicationService{AbstractSpecializedService: servutils.AbstractSpecializedService{
			ManyToMany: map[string][]map[string]interface{}{},
			OneToMany:  map[string][]map[string]interface{}{},
		},
			Sch: schemaName,
		}
	}
}

func (s *PublicationService) Entity() utils.SpecializedServiceInfo {
	return s.Sch
}

func (s *PublicationService) VerifyDataIntegrity(record map[string]interface{}, tablename string) (map[string]interface{}, error, bool) {
	fmt.Println("MAJOR CONF", s.Sch.HasField("major_conference"))
	if s.Sch.HasField("major_conference") {
		ok := record["major_conference"]
		isNotFound := true
		if res, err := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.MajorConference.Name, map[string]interface{}{}, false); err == nil && len(res) > 0 {
			for _, r := range res {
				fmt.Println(record, strings.ToUpper(utils.GetString(record, "conference_acronym")), "//", strings.ToUpper(utils.GetString(r, "name")), strings.Contains(strings.ToUpper(utils.GetString(record, "conference_accronym")), strings.ToUpper(utils.GetString(r, "name"))))
				if strings.Contains(strings.ToUpper(utils.GetString(record, "conference_acronym")), strings.ToUpper(utils.GetString(r, "name"))) {
					ok = true
					isNotFound = false
					break
				}
			}
		}
		if isNotFound {
			ok = false
		}
		record["major_conference"] = ok
		if ok == true {
			record["reread"] = 1
		}
	}
	return s.AbstractSpecializedService.VerifyDataIntegrity(record, tablename)
}

func (s *PublicationService) SpecializedUpdateRow(results []map[string]interface{}, record map[string]interface{}) {
	if record["state"] != nil && record["state"] != "" {
		id := s.Sch.GetID()
		if sc, err := schema.GetSchema(s.Sch.Name); err == nil {
			id = sc.GetID()
		}
		for _, r := range results {
			m := map[string]interface{}{
				ds.SchemaDBField:                           id,
				ds.DestTableDBField:                        r[utils.SpecialIDParam],
				ds.RootID(models.PublicationStatusFR.Name): utils.GetString(record, "state"),
			}
			if res, _ := s.Domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(models.PublicationHistoryStatusFR.Name, m, false); len(res) == 0 {
				s.Domain.GetDb().ClearQueryFilter().CreateQuery(models.PublicationHistoryStatusFR.Name, m, func(s string) (string, bool) { return s, true })
			}
		}

	}
	s.AbstractSpecializedService.SpecializedUpdateRow(results, record)
}

func (s *PublicationService) GenerateQueryFilter(tableName string, innerestr ...string) (string, string, string, string) {
	return filter.NewFilterService(s.Domain).GetQueryFilter(tableName, s.Domain.GetParams().Copy(), false, innerestr...)
}
