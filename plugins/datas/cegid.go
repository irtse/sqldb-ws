package datas

import (
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/schema/models"
)

var CoCFR = models.SchemaModel{
	Name:     "competence_center",
	Label:    "competence centers",
	Category: "domain",
	CanOwned: true,
	IsEnum:   true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "nom", Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: false, Index: 0},
		{Name: "label", Label: "label", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 0},
		{Name: ds.RootID(ds.DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: ds.DBEntity.Name, Required: true, Index: 1, Label: "entité en relation"},
	},
}

var Axis = models.SchemaModel{
	Name:     "axis",
	Label:    "IRT professional axis",
	CanOwned: true,
	Category: "domain",
	Fields: []models.FieldModel{
		{Name: "code", Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Translatable: false, Readonly: false, Index: 0},
		{Name: "name", Type: models.VARCHAR.String(), Constraint: "unique", Required: false, Translatable: false, Readonly: true, Index: 1},
		{Name: "domain_code", Label: "code domaine", Type: models.VARCHAR.String(), Translatable: false, Required: false, Readonly: true, Index: 2},
		{Name: ds.RootID(ds.DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: ds.DBEntity.Name, Required: true, Index: 3, Label: "entité en relation"},
	},
}

var ProjectFR = models.SchemaModel{ // todo
	Name:     "project",
	Label:    "projects",
	CanOwned: true,
	Category: "global data",
	Fields: []models.FieldModel{
		{Name: "code", Label: "code", Type: models.VARCHAR.String(), Constraint: "unique", Translatable: false, Required: true, Readonly: true, Index: 0},
		{Name: "name", Type: models.VARCHAR.String(), Required: false, Constraint: "unique", Translatable: false, Readonly: true, Index: 1},
		{Name: "state", Type: models.VARCHAR.String(), Required: false, Default: models.STATEPENDING, Level: models.LEVELRESPONSIBLE, Index: 2},
		{Name: "project_task", Label: "lot projet", Type: models.VARCHAR.String(), Required: false, Readonly: true, Index: 3},
		{Name: "prj_start_date", Label: "date de début de projet", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 4},
		{Name: "prj_end_date", Label: "date de fin de projet", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 5},
		{Name: ds.RootID(Axis.Name), Label: "axe", Type: models.INTEGER.String(), ForeignTable: Axis.Name, Required: false, Index: 6},
		{Name: ds.RootID(ds.DBUser.Name), Label: "chef de projet", Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 7},
		{Name: ds.RootID(ds.DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: ds.DBEntity.Name, Required: true, Index: 8, Label: "entité en relation"},
	},
}

// should set up as json better than a go file...

var PublicationStatusFR = models.SchemaModel{
	Name:     "publication_status",
	Label:    "publication status",
	Category: "domain",
	IsEnum:   true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "nom", Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: false, Index: 0},
	},
}

var PublicationHistoryStatusFR = models.SchemaModel{
	Name:     "publication_status_history",
	Label:    "publication status history",
	Category: "domain",
	IsEnum:   true,
	Fields: []models.FieldModel{
		{Name: ds.RootID(PublicationStatusFR.Name), Label: "related publication status", Type: models.INTEGER.String(), ForeignTable: PublicationStatusFR.Name, Required: true, Index: 1},
		{Name: "update_date", Label: "status update", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Default: "CURRENT_TIMESTAMP", Index: 2},
		{Name: ds.SchemaDBField, Label: "related publication schema", Type: models.INTEGER.String(), ForeignTable: ds.DBSchema.Name, Required: true, Index: 3},
		{Name: ds.DestTableDBField, Label: "related publication id", Type: models.INTEGER.String(), Required: true, Index: 4},
	},
}

var PublicationTagsFR = models.SchemaModel{
	Name:     "publication_tags",
	Label:    "publication tags",
	Category: "domain",
	IsEnum:   true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "nom", Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: false, Index: 0},
	},
}

var publicationFields = []models.FieldModel{
	{Name: "name", Label: "intitulé de la publication", Translatable: false,
		Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: false, Index: 0},
	{Name: "state", Default: 1, Type: models.INTEGER.String(), ForeignTable: PublicationStatusFR.Name, Required: false, Readonly: true, Label: "statut de publication", Index: 1},
	{Name: "tags", Type: models.MANYTOMANYADD.String(), Required: false, Translatable: false,
		ForeignTable: PublicationTagsFR.Name, Index: 1, Label: "mots-clés"},
	{Name: "manager_" + ds.RootID(ds.DBUser.Name), Type: models.INTEGER.String(), Required: true, Translatable: false,
		ForeignTable: ds.DBUser.Name, Index: 1, Label: "responsable IRT de la publication"},
	{Name: "project_accronym", Type: models.INTEGER.String(), Required: true, Translatable: false,
		Index: 2, Label: "projet", ForeignTable: Project.Name},
	{Name: "axis", Type: models.INTEGER.String(), Required: true,
		Index: 3, Label: "axe", ForeignTable: Axis.Name},
	{Name: "competence_center", Type: models.INTEGER.String(), Required: true, Translatable: false,
		Index: 4, Label: "centre de compétence", ForeignTable: CoCFR.Name},
}

var ArticleFR = models.SchemaModel{
	Name:     "article",
	Label:    "newspaper articles/book chapters",
	Category: "publications",
	CanOwned: true,
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true, Translatable: false,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},
		{Name: "published", Label: "la publication est elle publiée dans un journal du premier quartile de ta discipline scientifique", Type: models.ENUMBOOLEAN.String(), Translatable: false, Required: false, Default: false, Readonly: false, Index: -6},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: ArticleAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "false", Required: false, Readonly: false, Index: 9},
		{Name: "media_name", Label: "nom du journal", Type: models.VARCHAR.String(), Required: true, Readonly: false, Translatable: false, Index: 10},
		{Name: "publishing_date", Label: "date objective de publication", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 12},
	}...),
}

var ArticleAffiliationAuthorsFR = models.SchemaModel{
	Name:     "article_affiliation_authors",
	Label:    "article affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Required: true, Index: 1, Translatable: false, Label: "auteurs", ForeignTable: ArticleAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 2},
		{Name: ds.RootID("article"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "article", Required: true, Index: 3},
	},
}

var ArticleAuthorsFR = models.SchemaModel{
	Name:     "article_authors",
	Label:    "article authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Type: models.INTEGER.String(), Translatable: false, ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("article_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "article_affiliation_authors", Required: true, Index: 2},
	},
}

var ConferenceFR = models.SchemaModel{
	Name:     "conference_presentation",
	Label:    "presentations with congress proceedings",
	CanOwned: true,
	Category: "publications",
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true, Translatable: false,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},
		{Name: "major_conference", Translatable: false, Label: "la conférence visée est-elle incontournable dans ton domaine scientifique ?", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false, Readonly: false, Index: -8},
		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: ConferenceAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "true", Required: false, Readonly: false, Index: 9},
		{Name: "conference_acronym", Translatable: false, Label: "acronyme de la conférence ", Type: models.VARCHAR.String(), Required: true, Readonly: false, Index: 11},
		{Name: "conference_name", Translatable: false, Label: "nom de la conférence", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 10},
		{Name: "conference_start_date", Label: "date de début de la conférence", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 12},
		{Name: "conference_end_date", Label: "date de fin de la conférence", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 13},
		{Name: "conference_city", Label: "ville de la conférence", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 14},
		{Name: "conference_country", Label: "pays de la conférence", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 15},
		{Name: "conference_link", Label: "lien de la conférence", Type: models.URL.String(), Required: false, Readonly: false, Index: 16},
	}...),
}

var ConferenceAffiliationAuthorsFR = models.SchemaModel{
	Name:     "conference_affiliation_authors",
	Label:    "conference affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Required: true, Index: 1, Label: "auteurs", ForeignTable: ConferenceAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("conference_presentation"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "conference_presentation", Required: true, Index: 3},
	},
}

var ConferenceAuthorsFR = models.SchemaModel{
	Name:     "conference_authors",
	Label:    "conference authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Translatable: false, Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("conference_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "conference_affiliation_authors", Required: true, Index: 2},
	},
}

var PresentationFR = models.SchemaModel{
	Name:     "presentation",
	Label:    "presentations without proofreading (workshop, CST, GDR)",
	Category: "publications",
	CanOwned: true,
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: PresentationAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "false", Required: false, Readonly: true, Index: 9},
		{Name: "conference_acronym", Translatable: false, Label: "acronyme de la conférence", Type: models.VARCHAR.String(), Required: true, Readonly: false, Index: 10},
		{Name: "conference_name", Translatable: false, Label: "nom de la conférence", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 12},
		{Name: "meeting_name", Translatable: false, Label: "nom du meeting", Type: models.VARCHAR.String(), Required: false, Readonly: true, Index: 12},
		{Name: "meeting_date", Label: "date du meeting", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 13},
	}...),
}

var PresentationAffiliationAuthorsFR = models.SchemaModel{
	Name:     "presentation_affiliation_authors",
	Label:    "presentation affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Translatable: false, Required: true, Index: 1, Label: "auteurs", ForeignTable: PresentationAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("presentation"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "presentation", Required: true, Index: 3},
	},
}

var PresentationAuthorsFR = models.SchemaModel{
	Name:     "presentation_authors",
	Label:    "presentation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Translatable: false, Label: "related user", Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("presentation_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "presentation_affiliation_authors", Required: true, Index: 2},
	},
}

var PosterFR = models.SchemaModel{
	Name:     "poster",
	Label:    "posters",
	CanOwned: true,
	Category: "publications",
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true, Translatable: false,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},
		{Name: "major_conference", Label: "la conférence visée est-elle incontournable dans ton domaine scientifique ?", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false, Readonly: false, Index: -8},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: PosterAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "false", Required: false, Readonly: false, Index: 9},
		{Name: "conference_acronym", Label: "acronyme de la conférence", Translatable: false, Type: models.VARCHAR.String(), Required: true, Readonly: false, Index: 10},
		{Name: "conference_name", Translatable: false, Label: "nom de la conférence", Type: models.VARCHAR.String(), Required: true, Readonly: false, Index: 11},
		{Name: "conference_start_date", Label: "date objective de publication", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 12},
		{Name: "conference_city", Translatable: false, Label: "ville de la conférence", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 14},
		{Name: "conference_country", Translatable: false, Label: "pays de la conférence", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 15},
		{Name: "conference_link", Translatable: false, Label: "lien de la conférence", Type: models.URL.String(), Required: false, Readonly: false, Index: 16},
	}...),
}

var PosterAffiliationAuthorsFR = models.SchemaModel{
	Name:     "poster_affiliation_authors",
	Label:    "poster affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Required: true, Index: 1, Label: "auteurs", ForeignTable: PosterAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("poster"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "poster", Required: true, Index: 3},
	},
}

var PosterAuthorsFR = models.SchemaModel{
	Name:     "poster_authors",
	Label:    "poster authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Translatable: false, Label: "related user", Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("poster_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "poster_affiliation_authors", Required: true, Index: 2},
	},
}

var HDRFR = models.SchemaModel{
	Name:     "research_authorization",
	CanOwned: true,
	Label:    "authorizations to direct research",
	Category: "publications",
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: HDRAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "false", Required: false, Readonly: false, Index: 9},
		{Name: "defense_date", Label: "date de soutenance de l'HDR", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 10},
	}...),
}

var HDRAffiliationAuthorsFR = models.SchemaModel{
	Name:     "research_authorization_affiliation_authors",
	Label:    "research authorization affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Translatable: false, Required: true, Index: 1, Label: "auteurs", ForeignTable: HDRAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("research_authorization"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "research_authorization", Required: true, Index: 3},
	},
}

var HDRAuthorsFR = models.SchemaModel{
	Name:     "research_authorization_authors",
	Label:    "research authorization authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Translatable: false, Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("research_authorization_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "research_authorization_affiliation_authors", Required: true, Index: 2},
	},
}

var ThesisFR = models.SchemaModel{
	Name:     "thesis",
	Label:    "theses",
	CanOwned: true,
	Category: "publications",
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: ThesisAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "true", Required: false, Readonly: false, Index: 9},
		{Name: "defense_date", Label: "date de soutenance de thèse", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 10},
		{Name: "director_" + ds.RootID(ds.DBUser.Name), Translatable: false, Type: models.LINKADD.String(), Required: true, ForeignTable: ds.DBUser.Name, Index: 11, Label: "directeur de thèse"},
		{Name: "co_supervisor_" + ds.RootID(ds.DBUser.Name), Translatable: false, Type: models.MANYTOMANYADD.String(), Required: false, ForeignTable: ThesisSupervisorAuthorsFR.Name, Index: 12, Label: "co-encadrant de thèse"},
		{Name: "t_start_date", Label: "date de début de thèse", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 13},
		{Name: "t_end_date", Label: "date de fin de thèse", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 14},
	}...),
}

var ThesisSupervisorAuthorsFR = models.SchemaModel{
	Name:     "thesis_supervisor_authors",
	Label:    "thesis supervisor authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Translatable: false, Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("thesis"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "thesis", Required: true, Index: 3},
	},
}

var ThesisAffiliationAuthorsFR = models.SchemaModel{
	Name:     "thesis_affiliation_authors",
	Label:    "thesis affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Translatable: false, Required: true, Index: 1, Label: "auteurs", ForeignTable: ThesisAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("thesis"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "thesis", Required: true, Index: 3},
	},
}

var ThesisAuthorsFR = models.SchemaModel{
	Name:     "thesis_authors",
	Label:    "thesis authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Translatable: false, Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("thesis_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "thesis_affiliation_authors", Required: true, Index: 2},
	},
}

var InternshipFR = models.SchemaModel{
	Name:     "internship",
	Label:    "internships",
	CanOwned: true,
	Category: "publications",
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: InternshipAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "false", Required: false, Readonly: true, Index: 9},
		{Name: "IRT_manager" + ds.RootID(ds.DBUser.Name), Type: models.INTEGER.String(), Translatable: false, Required: true, ForeignTable: ds.DBUser.Name, Index: 10, Label: "responsable IRT du stage"},
		{Name: "i_start_date", Label: "date de soutenance de stage", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 11},
		{Name: "i_end_date", Label: "date de fin de stage", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 12},
	}...),
}

var InternshipAffiliationAuthorsFR = models.SchemaModel{
	Name:     "internship_affiliation_authors",
	Label:    "internship affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Translatable: false, Required: true, Index: 1, Label: "auteurs", ForeignTable: InternshipAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("internship"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "internship", Required: true, Index: 3},
	},
}

var InternshipAuthorsFR = models.SchemaModel{
	Name:     "internship_authors",
	Label:    "internship authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Translatable: false, Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("internship_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "internship_affiliation_authors", Required: true, Index: 2},
	},
}

var DemoFR = models.SchemaModel{
	Name:     "demo",
	Label:    "demos",
	CanOwned: true,
	Category: "publications",
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: DemoAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "false", Required: false, Readonly: true, Index: 9},
		{Name: "meeting_name", Translatable: false, Label: "nom du meeting", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 10},
		{Name: "meeting_date", Translatable: false, Label: "date du meeting", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 11},
	}...),
}

var DemoAffiliationAuthorsFR = models.SchemaModel{
	Name:     "demo_affiliation_authors",
	Label:    "demo affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Translatable: false, Required: true, Index: 1, Label: "auteurs", ForeignTable: DemoAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("demo"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "demo", Required: true, Index: 3},
	},
}

var DemoAuthorsFR = models.SchemaModel{
	Name:     "demo_authors",
	Label:    "demo authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Translatable: false, Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("demo_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "demo_affiliation_authors", Required: true, Index: 2},
	},
}

var OtherPublicationFR = models.SchemaModel{
	Name:     "other_publication",
	Label:    "other publications",
	Category: "publications",
	CanOwned: true,
	Fields: append(publicationFields, []models.FieldModel{
		{Name: "is_awarded", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false,
			Index: -20, Label: "la production a-t-elle fait l'objet d'une dsitinction ?"},
		{Name: "finalized_publication", Type: models.UPLOAD.String(), Required: true,
			Index: -10, Label: "publication finalisée", Subsection: "acte de publication"},
		{Name: "effective_publishing_date", Label: "date effective de publication", Type: models.TIMESTAMP.String(), Required: true, Readonly: false, Index: -9, Subsection: "acte de publication"},
		{Name: "major_conference", Label: "la conférence visée est-elle incontournable dans ton domaine scientifique ?", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false, Readonly: false, Index: -8},
		{Name: "conference_name", Translatable: false, Label: "nom de la conférence", Type: models.VARCHAR.String(), Required: false, Default: false, Readonly: false, Index: -7},
		{Name: "published", Label: "la publication est elle publiée dans un journal du premier quartile de ta discipline scientifique", Type: models.ENUMBOOLEAN.String(), Required: false, Default: false, Readonly: false, Index: -6},

		{Name: "authors", Type: models.ONETOMANY.String(), Required: true, Index: 8, Label: "auteurs  et affiliations", ForeignTable: OtherPublicationAffiliationAuthorsFR.Name},
		{Name: "reread", Label: "publication soumise à une revue par les pairs. En cas de doute, contacter la DRAI", Type: models.ENUMBOOLEAN.String(), Default: "false", Required: false, Readonly: false, Index: 9},
		{Name: "publishing_date", Label: "date objective de publication", Type: models.TIMESTAMP.String(), Required: false, Readonly: false, Index: 10},
	}...),
}

var OtherPublicationAffiliationAuthorsFR = models.SchemaModel{
	Name:     "other_publication_affiliation_authors",
	Label:    "other publication affiliation authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "authors", Type: models.MANYTOMANYADD.String(), Translatable: false, Required: true, Index: 1, Label: "auteurs", ForeignTable: OtherPublicationAuthorsFR.Name},
		{Name: "affiliation", Label: "affiliation", Translatable: false, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: ds.RootID("other_publication"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "other_publication", Required: true, Index: 3},
	},
}

var OtherPublicationAuthorsFR = models.SchemaModel{
	Name:     "other_publication_authors",
	Label:    "other publication authors",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Translatable: false, Required: true, Index: 1},
		{Name: ds.RootID(ds.DBUser.Name), Label: "related user", Translatable: false, Type: models.INTEGER.String(), ForeignTable: ds.DBUser.Name, Required: true, Index: 1},
		{Name: ds.RootID("other_publication_affiliation_authors"), Label: "related publication", Type: models.INTEGER.String(), ForeignTable: "other_publication_affiliation_authors", Required: true, Index: 2},
	},
}

var ProofreadingStatus = models.SchemaModel{
	Name:     "proofreading_status",
	Label:    "proofreading status",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Required: true, Index: 1},
	},
}

var MajorConference = models.SchemaModel{
	Name:     "major_conference",
	Label:    "major conference",
	Category: "publications",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "name", Label: "name", Type: models.VARCHAR.String(), Required: true, Index: 1},
	},
}
