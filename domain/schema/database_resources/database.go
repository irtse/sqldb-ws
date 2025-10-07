package database

import (
	"sqldb-ws/domain/schema/models"
	"strings"
)

/*
DB ROOT are all the ROOT database table needed in our generic API. They are restricted to modification
and can be impacted by a specialized service at DOMAIN level.
Their declarations is based on our Entity terminology, to help us in coding.
*/
// DBSchema express a table in the database, it's a template for a table
var DBSchema = models.SchemaModel{
	Name:     RootName("schema"),
	Label:    "templates",
	Category: "template",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Translatable: false, Level: models.LEVELRESPONSIBLE, Index: 0},
		{Name: models.LABELKEY, Type: models.BIGVARCHAR.String(), Required: true, Readonly: true, Index: 1},
		{Name: "category", Type: models.BIGVARCHAR.String(), Required: false, Default: "general", Readonly: true, Index: 2},
		{Name: "fields", Type: "onetomany", ForeignTable: RootName("schema_column"), Required: false, Index: 3},
		{Name: "can_owned", Label: "can be owned by a user", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 4},
		{Name: "is_enum", Label: "is a name list", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 5},
		{Name: "redirect_view_id_on_delete", Label: "redirect view id on deletion", Type: models.INTEGER.String(), Required: false, Default: false, Index: 6},
	},
}

// DBSchemaField express a column in a table, it's a template for a column
var DBSchemaField = models.SchemaModel{
	Name:     RootName("schema_column"),
	Label:    "template fields",
	Category: "template",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: true, Readonly: true, Index: 0},
		{Name: models.LABELKEY, Type: models.BIGVARCHAR.String(), Required: true, Index: 1},
		{Name: models.TYPEKEY, Type: models.VARCHAR.String(), Required: true, Index: 2},
		{Name: "description", Type: models.TEXT.String(), Required: false, Index: 3},
		{Name: "placeholder", Type: models.VARCHAR.String(), Required: false, Index: 4},
		{Name: "default_value", Type: models.BIGVARCHAR.String(), Required: false, Index: 5, Label: "default"},
		{Name: "index", Type: models.INTEGER.String(), Required: true, Default: 1, Index: 6},
		{Name: "in_resume", Type: models.VARCHAR.String(), Required: false, Index: 6},
		{Name: "subsection", Type: models.VARCHAR.String(), Required: false, Index: 6},
		{Name: "readonly", Type: models.BOOLEAN.String(), Required: true, Index: 7},
		{Name: "required", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 8},
		{Name: "read_level", Type: models.ENUMLEVEL.String(), Required: false, Default: models.LEVELNORMAL, Index: 9},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Index: 10, Label: "binded to template"},
		{Name: "constraints", Type: models.BIGVARCHAR.String(), Required: false, Level: models.LEVELRESPONSIBLE, Index: 11},
		{Name: "link_id", Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Index: 12, Label: "linked to"},
		{Name: "hidden", Type: models.BOOLEAN.String(), Default: false, Required: false, Index: 13, Label: "is hidden"},
		{Name: "translatable", Type: models.BOOLEAN.String(), Default: true, Required: false, Index: 14, Label: "is translatable"},
		{Name: "transform_function", Type: models.ENUMTRANSFORM.String(), Required: false, Index: 15, Label: "transformation function"},
		{Name: "group_by", Type: models.VARCHAR.String(), Required: false, Index: 16, Label: "group by"},
	},
}

// DBPermission express a permission in the database, ex: create, update, delete, read on a table
var DBPermission = models.SchemaModel{
	Name:     RootName("permission"),
	Label:    "permissions",
	Category: "role & permission",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Index: 0},
		{Name: models.CREATEPERMS, Type: models.BOOLEAN.String(), Required: true, Index: 1},
		{Name: models.UPDATEPERMS, Type: models.BOOLEAN.String(), Required: true, Index: 2},
		{Name: models.DELETEPERMS, Type: models.BOOLEAN.String(), Required: true, Index: 3},
		{Name: models.READPERMS, Type: models.ENUMLEVELCOMPLETE.String(), Required: false, Default: models.LEVELNORMAL, Index: 4},
	},
}

// DBRole express a role in the database, ex: admin, user, guest with a set of permissions
var DBRole = models.SchemaModel{
	Name:     RootName("role"),
	Label:    "roles",
	IsEnum:   true,
	Category: "role & permission",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Index: 0},
		{Name: "description", Type: models.TEXT.String(), Required: false, Index: 1},
	},
}

// DBRolePermission express a role permission attribution in the database
var DBRolePermission = models.SchemaModel{
	Name:     RootName("role_permission"),
	Label:    "permission role attributions",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBRole.Name), Type: models.INTEGER.String(), ForeignTable: DBRole.Name, Required: true, Readonly: true, Index: 0, Label: "role"},
		{Name: RootID(DBPermission.Name), Type: models.INTEGER.String(), ForeignTable: DBPermission.Name, Required: true, Readonly: true, Index: 1, Label: "permission"},
	},
}

// DBEntity express an entity in the database, ex: user, task, project
var DBEntity = models.SchemaModel{
	Name:     RootName("entity"),
	Label:    "entities",
	Category: "entity",
	CanOwned: true,
	IsEnum:   true,
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: true, Readonly: true, Index: 0},
		{Name: "description", Type: models.TEXT.String(), Required: false, Index: 1},
		{Name: "parent_id", Type: models.INTEGER.String(), ForeignTable: RootName("entity"), Required: false, Index: 2, Label: "parent entity"},
	},
}

// DBUser express a user in the database, with email, password, token, super_admin
var DBUser = models.SchemaModel{
	Name:     RootName("user"),
	Label:    "users",
	Category: "user",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Index: 0},
		{Name: "email", Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Index: 1},
		{Name: "password", Type: models.TEXT.String(), Required: false, Default: "", Level: models.LEVELRESPONSIBLE, Index: 2},
		{Name: "token", Type: models.TEXT.String(), Required: false, Default: "", Level: models.LEVELRESPONSIBLE, Index: 3},
		{Name: "super_admin", Type: models.BOOLEAN.String(), Required: false, Default: false, Level: models.LEVELRESPONSIBLE, Index: 4},
		{Name: "code", Type: models.VARCHAR.String(), Required: false, Readonly: true, Level: models.LEVELRESPONSIBLE, Index: 5},
	},
}

var DBEmailTemplate = models.SchemaModel{
	Name:     RootName("email_template"),
	Label:    "email templates",
	Category: "email",
	Fields: []models.FieldModel{
		{Name: "subject", Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Index: 1},
		{Name: "template", Type: models.TEXT.String(), Required: true, Index: 2},
		{Name: "signature", Type: models.TEXT.String(), Required: true, Index: 2},
		{Name: "waiting_response", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 3},
		{Name: "to_map_" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 4},
		{Name: "redirected_on", Type: models.VARCHAR.String(), Required: false, Index: 5},
		{Name: "generate_task", Type: models.BOOLEAN.String(), Required: false, Index: 6},
		{Name: "force_file_attached", Type: models.BOOLEAN.String(), Required: false, Index: 6},

		{Name: "is_response_valid", Type: models.BOOLEAN.String(), Required: false, Index: 6},
		{Name: "is_response_refused", Type: models.BOOLEAN.String(), Required: false, Index: 6},

		{Name: "action_on_response", Type: models.VARCHAR.String(), Required: false, Readonly: true, Label: "action on response", Index: 7},
		{Name: RootID(DBSchema.Name) + "_on_response", Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Readonly: true, Label: "template to modify on response", Index: 8},
		{Name: "body_on_true_response", Type: models.VARCHAR.String(), Required: false, Readonly: true, Label: "body sended on valid response", Index: 9},
		{Name: "body_on_false_response", Type: models.VARCHAR.String(), Required: false, Readonly: true, Label: "body sended on unvalid response", Index: 10},
	},
}

var DBEmailSended = models.SchemaModel{
	Name:     RootName("email_sended"),
	Label:    "email sended",
	Category: "email",
	Fields: []models.FieldModel{
		{Name: "to_email", Type: models.MANYTOMANYADD.String(), Label: "sent to", ForeignTable: RootName("email_sended_user"), Required: true, Readonly: true, Index: 0},
		{Name: "from_email", Type: models.INTEGER.String(), Label: "sent from", ForeignTable: DBUser.Name, Required: true, Readonly: false, Index: 1},
		{Name: "subject", Type: models.VARCHAR.String(), Required: true, Readonly: false, Index: 2},
		{Name: "file_attached", Type: models.UPLOAD_MULTIPLE.String(), Required: false, Readonly: false, Label: "file attached", Index: 3},
		{Name: "content", Type: models.HTML.String(), Label: "message", Required: false, Index: 4},
		{Name: RootID(DBEmailTemplate.Name), Type: models.INTEGER.String(), ForeignTable: DBEmailTemplate.Name, Required: true, Readonly: true, Label: "email attached", Index: 5},
		{Name: "code", Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Index: 6},
		{Name: "mapped_with" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 7},
		{Name: "mapped_with" + RootID("dest_table"), Type: models.INTEGER.String(), Required: true, Readonly: true, Label: "template attached", Index: 8},
		{Name: RootID("dest_table") + "_on_response", Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "data to modify on response", Index: 9},
	},
}

var DBEmailSendedUser = models.SchemaModel{
	Name:     RootName("email_sended_user"),
	Label:    "email sended to user",
	Category: "email",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: false, Readonly: true, Index: 0},
		{Name: RootID(DBEmailSended.Name), Type: models.INTEGER.String(), ForeignTable: DBEmailSended.Name, Required: true, Readonly: true, Index: 1},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Readonly: false, Index: 2},
	},
}

var DBEmailResponse = models.SchemaModel{
	Name:     RootName("email_response"),
	Label:    "email responses",
	Category: "email",
	Fields: []models.FieldModel{
		{Name: "got_response", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 0},
		{Name: "comment", Type: models.VARCHAR.String(), Required: false, Index: 1},
		{Name: RootID(DBEmailSended.Name), Type: models.INTEGER.String(), ForeignTable: DBEmailSended.Name, Required: true, Readonly: true, Label: "email attached", Index: 2},
	},
}

var DBEmailList = models.SchemaModel{
	Name:     RootName("email_list"),
	Label:    "email listing",
	Category: "email",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: false, Readonly: true, Index: 0},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Readonly: false, Index: 2},
	},
}

var DBTrigger = models.SchemaModel{
	Name:     RootName("triggers"),
	Label:    "triggers",
	Category: "trigger",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Constraint: "unique", Required: true, Readonly: true, Index: 0},
		{Name: "description", Type: models.TEXT.String(), Required: false, Readonly: true, Index: 0},
		{Name: "type", Type: models.ENUMTRIGGER.String(), Required: true, Readonly: true, Index: 1},
		{Name: "mode", Type: models.ENUMMODE.String(), Required: true, Readonly: true, Index: 1},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 3},
		{Name: "on_write", Type: models.BOOLEAN.String(), Required: true, Readonly: false, Default: false, Label: "on creation", Index: 2},
		{Name: "on_update", Type: models.BOOLEAN.String(), Required: true, Readonly: false, Default: false, Label: "on update", Index: 3},
	},
}

var DBTriggerCondition = models.SchemaModel{
	Name:     RootName("triggers_condition"),
	Label:    "triggers conditions",
	Category: "trigger",
	Fields: []models.FieldModel{
		{Name: "value", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 0},
		{Name: "not_null", Type: models.BOOLEAN.String(), Required: false, Readonly: false, Default: false, Index: 1},

		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template to check condition", Index: 2},
		{Name: RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: true, Readonly: true, Label: "field to check condition", Index: 3},
		{Name: RootID(DBTrigger.Name), Type: models.INTEGER.String(), ForeignTable: DBTrigger.Name, Required: true, Readonly: true, Label: "related trigger", Index: 4},
	},
}

var DBTriggerDestination = models.SchemaModel{
	Name:     RootName("triggers_destination"),
	Label:    "triggers destinations",
	Category: "trigger",
	Fields: []models.FieldModel{
		{Name: "value", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 0},
		{Name: "is_own", Type: models.BOOLEAN.String(), Required: false, Readonly: false, Index: 1},
		{Name: "from_" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Readonly: true, Label: "template to extract value modification", Index: 2},
		{Name: "from_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field  to extract value modification", Index: 3},
		{Name: "from_compare_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field  to extract value modification", Index: 3},

		{Name: RootID(DBTrigger.Name), Type: models.INTEGER.String(), ForeignTable: DBTrigger.Name, Required: true, Readonly: true, Label: "related trigger", Index: 6},
	},
}

var DBTriggerRule = models.SchemaModel{
	Name:     RootName("triggers_rule"),
	Label:    "triggers rules",
	Category: "trigger",
	Fields: []models.FieldModel{
		{Name: "value", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 0},

		{Name: "from_" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Readonly: true, Label: "template to extract value modification", Index: 2},
		{Name: "from_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field  to extract value modification", Index: 3},

		{Name: "to_" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template to apply modification", Index: 5},
		{Name: "to_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: true, Readonly: true, Label: "field to apply modification", Index: 6},
		{Name: RootID(DBTrigger.Name), Type: models.INTEGER.String(), ForeignTable: DBTrigger.Name, Required: true, Readonly: true, Label: "related trigger", Index: 6},
	},
}

var DBFieldAutoFill = models.SchemaModel{
	Name:     RootName("field_autofill"),
	Label:    "auto-fill fields",
	Category: "schema",
	Fields: []models.FieldModel{
		{Name: "value", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 0},
		{Name: "first_own", Type: models.BOOLEAN.String(), Label: "first of our data", Required: false, Readonly: false, Default: false, Index: 1},

		{Name: "from_" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Readonly: true, Label: "template to extract value modification", Index: 2},
		{Name: "from_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field  to extract value modification", Index: 3},
		{Name: "from_" + RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 4},

		{Name: RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: true, Readonly: true, Label: "field to check condition", Index: 5},
	},
}

var DBFieldRule = models.SchemaModel{
	Name:     RootName("field_rule"),
	Label:    "field rules",
	Category: "schema",
	Fields: []models.FieldModel{
		{Name: "value", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 0},
		{Name: "min", Type: models.BOOLEAN.String(), Default: false, Readonly: false, Index: 0},
		{Name: "max", Type: models.BOOLEAN.String(), Default: false, Readonly: false, Index: 0},
		{Name: "starting_rule", Type: models.BOOLEAN.String(), Required: true, Readonly: false, Index: 0},
		{Name: "operator", Type: models.ENUMOPERATOR.String(), Required: false, Index: 1},
		{Name: "separator", Type: models.ENUMSEPARATOR.String(), Required: false, Index: 2},

		{Name: "from_" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Readonly: true, Label: "template to extract value modification", Index: 2},
		{Name: "from_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field  to extract value modification", Index: 3},
		{Name: "from_" + RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 4},

		{Name: RootID("field_rule"), Type: models.INTEGER.String(), ForeignTable: RootID("field_rule"), Required: false, Readonly: true, Label: "dependents to another rule", Index: 4},
		{Name: "related_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "related field", Index: 5},
		{Name: RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field to bind rule", Index: 6},
	},
}

var DBFieldCondition = models.SchemaModel{
	Name:     RootName("field_condition"),
	Label:    "field conditions",
	Category: "schema",
	Fields: []models.FieldModel{
		{Name: "value", Type: models.VARCHAR.String(), Required: false, Readonly: false, Index: 0}, // if not then use schemafield value.
		{Name: "not_null", Type: models.BOOLEAN.String(), Required: false, Readonly: false, Default: false, Index: 1},

		{Name: "from_" + RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Readonly: true, Label: "template to extract value modification", Index: 2},
		{Name: "from_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field  to extract value modification", Index: 3},

		{Name: RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Readonly: true, Label: "field to check condition", Index: 4},
	},
}

// Note rules : HIERARCHY IS NOT INNER ROLE. HIERARCHY DEFINE MASTER OF AN ENTITY OR A USER. IT'S AN AUTO WATCHER ON USER ASSIGNEE TASK.
var DBHierarchy = models.SchemaModel{
	Name:     RootName("hierarchy"),
	Label:    "hierarchies",
	Category: "user",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "parent_" + RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: true, Index: 0, Label: "hierarchical user"},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Index: 1, Label: "user with hierarchy"},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Index: 2, Label: "entity with hierarchy"},
		{Name: models.STARTKEY, Type: models.TIMESTAMP.String(), Required: false, Default: "CURRENT_TIMESTAMP", Index: 3},
		{Name: models.ENDKEY, Type: models.TIMESTAMP.String(), Required: false, Index: 4},
	},
}

// DBEntityAttribution express an entity attribution in the database
var DBEntityUser = models.SchemaModel{
	Name:     RootName("entity_user"),
	Label:    "entity user attributions",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: true, Readonly: true, Index: 0, Label: "user"},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: true, Readonly: true, Index: 1, Label: "entity"},
		{Name: models.STARTKEY, Type: models.TIMESTAMP.String(), Required: false, Default: "CURRENT_TIMESTAMP", Index: 2},
		{Name: models.ENDKEY, Type: models.TIMESTAMP.String(), Required: false, Index: 3},
	},
}

// DBRoleAttribution express a role attribution in the database
var DBRoleAttribution = models.SchemaModel{
	Name:     RootName("role_attribution"),
	Label:    "role attributions",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Readonly: true, Index: 0, Label: "user"},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Readonly: true, Index: 1, Label: "entity"},
		{Name: RootID(DBRole.Name), Type: models.INTEGER.String(), ForeignTable: DBRole.Name, Required: true, Readonly: true, Index: 2, Label: "role"},
		{Name: models.STARTKEY, Type: models.TIMESTAMP.String(), Required: false, Default: "CURRENT_TIMESTAMP", Index: 3},
		{Name: models.ENDKEY, Type: models.TIMESTAMP.String(), Required: false, Index: 4},
	},
}

// DBWorkflow express a workflow in the database, a workflow is a set of steps to achieve a request
var DBWorkflow = models.SchemaModel{
	Name:     RootName("workflow"),
	Label:    "workflows",
	Category: "workflow",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Constraint: "unique", Type: models.VARCHAR.String(), Required: true, Readonly: true, Index: 0},
		{Name: "description", Type: models.BIGVARCHAR.String(), Required: false, Index: 1},
		{Name: "is_meta", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 2, Label: "is a meta request"},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template entry", Index: 3},
		{Name: "steps", Type: "onetomany", ForeignTable: RootName("workflow_schema"), Required: false, Index: 4},
		{Name: "view_" + RootID(DBFilter.Name), Type: models.INTEGER.String(), ForeignTable: DBFilter.Name, Required: false, Label: "filter to apply on step", Index: 5, Hidden: true},
	},
}

// DBWorkflowSchema express a workflow schema in the database, a workflow schema is a step in a workflow
var DBWorkflowSchema = models.SchemaModel{
	Name:     RootName("workflow_schema"),
	Label:    "workflow schema attributions",
	Category: "",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: true, Constraint: "unique", Readonly: true, Index: 0},
		{Name: "description", Type: models.TEXT.String(), Required: false, Index: 1},
		{Name: "index", Type: models.INTEGER.String(), Required: true, Default: 1, Index: 2},
		{Name: "urgency", Type: models.ENUMURGENCY.String(), Required: false, Default: models.LEVELNORMAL, Index: 3},
		{Name: "priority", Type: models.ENUMURGENCY.String(), Required: false, Default: models.LEVELNORMAL, Index: 4},
		{Name: "optionnal", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 5},
		{Name: "hub", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 6},
		{Name: RootID(DBWorkflow.Name), Type: models.INTEGER.String(), ForeignTable: DBWorkflow.Name, Required: true, Readonly: true, Label: "workflow attached", Index: 7},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 8},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Label: "user assignee", Index: 9},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Label: "entity assignee", Index: 10},
		{Name: "wrapped_" + RootID(DBWorkflow.Name), Type: models.INTEGER.String(), ForeignTable: DBWorkflow.Name, Required: false, Readonly: true, Label: "wrapping workflow", Index: 11},
		{Name: "before_hierarchical_validation", Type: models.BOOLEAN.String(), Required: false, Readonly: false, Label: "must have a before hierarchical validation", Index: 12},
		{Name: "custom_progressing_status", Type: models.VARCHAR.String(), Required: false, Readonly: true, Label: "rename of the pending status", Index: 13},
		{Name: "view_" + RootID(DBFilter.Name), Type: models.INTEGER.String(), ForeignTable: DBFilter.Name, Required: false, Label: "filter to apply on step", Index: 10, Hidden: true},
		{Name: "readonly_not_assignee", Type: models.BOOLEAN.String(), Required: false, Default: false, Label: "readonly for not assignee", Index: 11, Hidden: true},
		{Name: "assign_to_creator", Type: models.BOOLEAN.String(), Required: false, Default: false, Label: "assign to creator", Index: 12, Hidden: true},

		{Name: "override_state_completed", Type: models.VARCHAR.String(), Required: false, Index: 14},
		{Name: "override_state_dismiss", Type: models.VARCHAR.String(), Required: false, Index: 15},
		{Name: "override_state_refused", Type: models.VARCHAR.String(), Required: false, Index: 16},
	},
}

// TODO RELATE FILTER TO TASK IF ONE

// DBRequest express a request in the database, a request is a set of tasks to achieve a goal
var DBRequest = models.SchemaModel{
	Name:     RootName("request"),
	Label:    "requests",
	Category: "request",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.TEXT.String(), Required: true, Readonly: true, Index: 0},
		{Name: "state", Type: models.VARCHAR.String(), Required: false, Default: models.STATEPENDING, Level: models.LEVELRESPONSIBLE, Index: 1},
		{Name: "is_close", Type: models.BOOLEAN.String(), Required: false, Default: false, Level: models.LEVELRESPONSIBLE, Index: 2},
		{Name: "current_index", Type: models.FLOAT8.String(), Required: false, Default: 0, Index: 3},
		{Name: "closing_date", Type: models.TIMESTAMP.String(), Required: false, Readonly: true, Level: models.LEVELRESPONSIBLE, Index: 5},
		{Name: RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 6},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 7},
		{Name: RootID(DBWorkflow.Name), Type: models.INTEGER.String(), ForeignTable: DBWorkflow.Name, Required: false, Label: "request type", Index: 8},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Label: "created by", Index: 9},
		{Name: "is_meta", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 10, Hidden: true},
		{Name: "closing_comment", Type: models.HTML.String(), Required: false, Index: 13, Hidden: true},
	},
}

// DBWorkflow express a workflow in the database, a workflow is a set of steps to achieve a request
var DBConsent = models.SchemaModel{
	Name:     RootName("consent"),
	Label:    "consents",
	Category: "consent",
	IsEnum:   true,
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Constraint: "unique", Type: models.VARCHAR.String(), Required: true, Readonly: true, Index: 0},
		{Name: "optionnal", Type: models.BOOLEAN.String(), Required: true, Default: false, Index: 1},
		{Name: "on_create", Type: models.BOOLEAN.String(), Required: true, Default: true, Index: 2},
		{Name: "on_update", Type: models.BOOLEAN.String(), Required: true, Default: true, Index: 3},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 4},
	},
}

var DBConsentResponse = models.SchemaModel{
	Name:     RootName("consent_response"),
	Label:    "consent responses",
	Category: "consent",
	IsEnum:   true,
	Fields: []models.FieldModel{
		{Name: "is_consenting", Label: "consentant", Type: models.BOOLEAN.String(), Required: true, Readonly: false, Index: 0},
		{Name: RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 1},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 2},
		{Name: RootID(DBConsent.Name), Type: models.INTEGER.String(), ForeignTable: DBConsent.Name, Required: true, Readonly: true, Label: "consent template attached", Index: 3, Hidden: true},
	},
}

// DBTask express a task in the database, a task is an activity to achieve a step in a request
var DBTask = models.SchemaModel{
	Name:     RootName("task"),
	Label:    "activities",
	Category: "request",
	Fields: []models.FieldModel{
		{Name: RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 0},
		{Name: models.NAMEKEY, Label: "task to be done", Type: models.TEXT.String(), Required: true, Readonly: true, Index: 1},
		{Name: "description", Type: models.BIGVARCHAR.String(), Required: false, Index: 11},
		{Name: "state", Type: models.ENUMSTATE.String(), Required: false, Default: models.STATEPENDING, Index: 2},
		{Name: "is_close", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 3, Hidden: true},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Readonly: true, Label: "assigned to user", Index: 3},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Readonly: true, Label: "assigned to entity", Index: 4},
		{Name: "urgency", Type: models.ENUMURGENCY.String(), Required: false, Default: models.LEVELNORMAL, Readonly: true, Index: 5},
		{Name: "priority", Type: models.ENUMURGENCY.String(), Required: false, Default: models.LEVELNORMAL, Readonly: true, Index: 6},
		{Name: "closing_date", Type: models.TIMESTAMP.String(), Required: false, Readonly: true, Index: 7},
		{Name: "closing_by" + RootID(DBUser.Name), Type: models.TIMESTAMP.String(), Required: false, Readonly: true, Index: 8},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 9},
		{Name: RootID(DBRequest.Name), Type: models.INTEGER.String(), ForeignTable: DBRequest.Name, Required: true, Readonly: true, Label: "request attached", Index: 10},
		{Name: RootID(DBWorkflowSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBWorkflowSchema.Name, Required: false, Hidden: true, Readonly: true, Label: "workflow attached", Index: 11},
		{Name: "nexts", Type: models.BIGVARCHAR.String(), Required: false, Default: "all", Hidden: true, Index: 12},
		{Name: "meta_" + RootID(DBRequest.Name), Type: models.INTEGER.String(), ForeignTable: DBRequest.Name, Required: false, Hidden: true, Readonly: true, Label: "meta request attached", Index: 13},
		{Name: "binded_dbtask", Type: models.INTEGER.String(), ForeignTable: "dbtask", Required: false, Readonly: true, Label: "binded task", Hidden: true, Index: 14},
		{Name: "closing_comment", Type: models.HTML.String(), Required: false, Index: 13, Hidden: true},

		{Name: "override_state_completed", Type: models.VARCHAR.String(), Required: false, Index: 14},
		{Name: "override_state_dismiss", Type: models.VARCHAR.String(), Required: false, Index: 15},
		{Name: "override_state_refused", Type: models.VARCHAR.String(), Required: false, Index: 16},

		{Name: "opening_date", Type: models.TIMESTAMP.String(), Required: false, Readonly: true, Index: 7},
	},
}

var DBComment = models.SchemaModel{
	Name:     RootName("comment"),
	Label:    "commentaries",
	Category: "",
	Fields: []models.FieldModel{
		{Name: "content", Type: models.VARCHAR.String(), Required: true, Readonly: true, Index: 0},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Readonly: true, Label: "comment by", Level: models.LEVELRESPONSIBLE, Index: 2},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 3},
		{Name: RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 4}, // reference to a table if needed
	},
}

// DBFilter express a filter in the database, a filter is a set of conditions to filter a view on a table
var DBFilter = models.SchemaModel{
	Name:     RootName("filter"),
	Label:    "filters",
	Category: "",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: true, Index: 0},
		{Name: "is_view", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 1},
		{Name: "is_selected", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 2},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Index: 3},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Index: 4},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Index: 5},
		{Name: "elder", Type: models.ENUMLIFESTATE.String(), Required: false, Default: "all", Index: 6},
		{Name: "dashboard_restricted", Type: models.BOOLEAN.String(), Required: true, Default: false, Index: 7},
		{Name: "hidden", Type: models.BOOLEAN.String(), Required: false, Default: true, Index: 8},
	},
}

// DBFilterField express a filter field in the database, a filter field is a condition to filter a view on a table
var DBFilterField = models.SchemaModel{
	Name:     RootName("filter_field"),
	Label:    "filter fields",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Index: 0},
		{Name: "value", Type: models.BIGVARCHAR.String(), Required: false, Index: 1},
		{Name: "name", Type: models.VARCHAR.String(), Required: false, Index: 1},

		{Name: "operator", Type: models.ENUMOPERATOR.String(), Required: false, Index: 2},
		{Name: "separator", Type: models.ENUMSEPARATOR.String(), Required: false, Index: 3},
		{Name: "dir", Type: models.BIGVARCHAR.String(), Required: false, Index: 4},
		{Name: "index", Type: models.INTEGER.String(), Required: false, Default: 1, Index: 5},
		{Name: "width", Type: models.DECIMAL.String(), Required: false, Index: 6},
		{Name: "is_own", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 7},

		{Name: "force_not_readonly", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 7},

		{Name: "is_task_concerned", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 8},
		{Name: RootID(DBFilter.Name), Type: models.INTEGER.String(), ForeignTable: DBFilter.Name, Required: false, Index: 9},
	},
}

// DBDashboardElement express a dashboard in the database, a dashboard is a set of views on a table
var DBDashboard = models.SchemaModel{
	Name:     RootName("dashboard"),
	Label:    "dashboards",
	Category: "",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: true, Index: 0},
		{Name: "description", Type: models.BIGVARCHAR.String(), Required: false, Index: 1},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Index: 2},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Index: 3},
		{Name: "is_selected", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 4},
		{Name: "url", Type: models.VARCHAR.String(), Required: false, Default: false, Index: 5},
	},
}

// DBDashboardElement express a dashboard element in the database, a dashboard element is a view on a table with a filter
var DBDashboardElement = models.SchemaModel{
	Name:     RootName("dashboard_element"),
	Label:    "dashboard elements",
	Category: "",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: true, Index: 0},
		{Name: "description", Type: models.BIGVARCHAR.String(), Required: false, Index: 1},
		{Name: "type", Type: models.ENUMTIME.String(), Required: false, Index: 2},
		{Name: "X", Type: models.INTEGER.String(), ForeignTable: DBDashboardLabel.Name, Required: true, Index: 3},
		{Name: "Y", Type: models.VARCHAR.String(), ForeignTable: DBDashboardLabel.Name, Required: false, Index: 4},
		{Name: "Z", Type: models.VARCHAR.String(), ForeignTable: DBDashboardLabel.Name, Required: false, Index: 5},
		{Name: RootID(DBDashboardMathField.Name), Type: models.INTEGER.String(), ForeignTable: DBDashboardMathField.Name, Required: false, Index: 6},
		{Name: RootID(DBFilter.Name), Type: models.INTEGER.String(), ForeignTable: DBFilter.Name, Required: false, Index: 7},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Index: 8},                          // results if multiple must be ordered by
		{Name: "order_by_" + RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Index: 9}, // results if multiple must be ordered by
		{Name: RootID(DBDashboard.Name), Type: models.INTEGER.String(), ForeignTable: DBDashboard.Name, Required: true, Index: 10},
	},
}

// DBDashboardMathField express a dashboard math field in the database, a dashboard math field is a math operation on a column
var DBDashboardLabel = models.SchemaModel{
	Name:     RootName("dashboard_math_field"),
	Label:    "dashboard math fields",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: false, Index: 1},
		{Name: RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Index: 2},
		{Name: "type", Type: models.VARCHAR.String(), Required: false, Index: 3},
	},
}

// DBDashboardMathField express a dashboard math field in the database, a dashboard math field is a math operation on a column
var DBDashboardMathField = models.SchemaModel{
	Name:     RootName("dashboard_math_field"),
	Label:    "dashboard math fields",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Index: 1},
		{Name: RootID(DBSchemaField.Name), Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: true, Index: 2},
		{Name: "column_math_func", Type: models.ENUMMATHFUNC.String(), Required: false, Index: 3}, // func applied on operation added on column value ex: COUNT
		{Name: "row_math_func", Type: models.VARCHAR.String(), Required: false, Index: 4},
	},
}

// DBView express a view in the database, a view is a set of fields to display on a table
var DBView = models.SchemaModel{
	Name:     RootName("view"),
	Label:    "views",
	Category: "view",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.VARCHAR.String(), Required: true, Constraint: "unique", Index: 0},
		{Name: "label", Type: models.VARCHAR.String(), Index: 0},
		{Name: "description", Type: models.BIGVARCHAR.String(), Required: false, Index: 1},
		{Name: "category", Type: models.VARCHAR.String(), Required: false, Index: 2},
		{Name: "index", Type: models.INTEGER.String(), Required: false, Default: 1, Index: 3},
		{Name: "indexable", Type: models.BOOLEAN.String(), Required: false, Default: true, Index: 4},
		{Name: "is_list", Type: models.BOOLEAN.String(), Required: false, Default: true, Index: 5},
		{Name: "shortcut_on_schema", Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Label: "is a shortcut on data", Required: false, Index: 6},
		{Name: "shortcut_on_main", Type: models.BOOLEAN.String(), Label: "is a shortcut on home", Required: false, Index: 7},
		{Name: "is_empty", Type: models.BOOLEAN.String(), Required: false, Default: false, Index: 8},
		{Name: "readonly", Type: models.BOOLEAN.String(), Required: true, Index: 9},
		{Name: "view_" + RootID(DBFilter.Name), Type: models.INTEGER.String(), ForeignTable: DBFilter.Name, Required: false, Index: 10},
		{Name: RootID(DBFilter.Name), Type: models.INTEGER.String(), ForeignTable: DBFilter.Name, Required: false, Index: 11},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Index: 12},
		{Name: "own_view", Type: models.BOOLEAN.String(), Required: false, Index: 13},
		{Name: "group_by", Type: models.INTEGER.String(), ForeignTable: DBSchemaField.Name, Required: false, Label: "group by", Index: 14},
		{Name: "only_super_admin", Type: models.BOOLEAN.String(), Required: false, Default: false, Level: models.LEVELADMIN, Index: 15},
	},
}

// DBViewAttribution express a view attribution in the database for a user or an entity
var DBViewSchema = models.SchemaModel{
	Name:     RootName("view_schema"),
	Label:    "view schemas",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBView.Name), Type: models.INTEGER.String(), ForeignTable: DBView.Name, Required: true, Index: 0},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Index: 1},
	},
}

// DBViewAttribution express a view attribution in the database for a user or an entity
var DBViewAttribution = models.SchemaModel{
	Name:     RootName("view_attribution"),
	Label:    "view attributions",
	Category: "",
	Fields: []models.FieldModel{
		{Name: RootID(DBView.Name), Type: models.INTEGER.String(), ForeignTable: DBView.Name, Required: true, Index: 0},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Index: 1},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Index: 2},
	},
}

// DBNotification express a notification in the database, a notification is a message to a user or an entity
var DBNotification = models.SchemaModel{
	Name:     RootName("notification"),
	Label:    "notifications",
	Category: "",
	Fields: []models.FieldModel{
		{Name: models.NAMEKEY, Type: models.TEXT.String(), Required: true, Index: 0},
		{Name: "description", Type: models.BIGVARCHAR.String(), Required: false, Index: 1},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Readonly: true, Label: "assigned user", Index: 2},
		{Name: RootID(DBEntity.Name), Type: models.INTEGER.String(), ForeignTable: DBEntity.Name, Required: false, Readonly: true, Label: "assigned entity", Index: 3},
		{Name: RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 4}, // reference to a table if needed
		{Name: "link_id", Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Readonly: true, Label: "template attached", Index: 5},
	},
}

// DBDataAccess express a data access in the database, a data access is a log of access to a table
var DBDataAccess = models.SchemaModel{
	Name:     RootName("data_access"),
	Label:    "data access",
	Category: "history",
	Fields: []models.FieldModel{
		{Name: "update", Type: models.BOOLEAN.String(), Required: false, Default: false, Readonly: true, Label: "updated", Index: 0},
		{Name: "write", Type: models.BOOLEAN.String(), Required: false, Default: false, Readonly: true, Label: "created", Index: 1},
		{Name: "access_date", Type: models.TIMESTAMP.String(), Required: false, Default: "CURRENT_TIMESTAMP", Readonly: true, Label: "access date", Index: 2},
		{Name: RootID("dest_table"), Type: models.INTEGER.String(), Required: false, Readonly: true, Label: "reference", Index: 3},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Readonly: true, Label: "template attached", Index: 4},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Readonly: true, Label: "related user", Index: 5},
	},
}

var DBDelegation = models.SchemaModel{
	Name:     RootName("delegation"),
	Label:    "delegation",
	Category: "users",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "delegated_" + RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: true, Index: 0, Label: "delegated to user"},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Index: 1,
			Level: models.LEVELADMIN, Label: "user with hierarchy"},
		{Name: models.STARTKEY, Type: models.TIMESTAMP.String(), Required: false, Default: "CURRENT_TIMESTAMP", Index: 2},
		{Name: models.ENDKEY, Type: models.TIMESTAMP.String(), Required: false, Index: 3},
		{Name: RootID(DBTask.Name), Type: models.INTEGER.String(), ForeignTable: DBTask.Name, Required: false, Index: 4, Label: "task delegated"},
		{Name: "all_tasks", Type: models.BOOLEAN.String(), Required: false, Index: 5, Label: "all tasks"},
	},
}

var DBShare = models.SchemaModel{
	Name:     RootName("share"),
	Label:    "sharings",
	Category: "user",
	CanOwned: true,
	Fields: []models.FieldModel{
		{Name: "shared_" + RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: true, Index: 0, Label: "shared to user"},
		{Name: RootID(DBUser.Name), Type: models.INTEGER.String(), ForeignTable: DBUser.Name, Required: false, Index: 1, Label: "user with hierarchy"},
		{Name: models.STARTKEY, Type: models.TIMESTAMP.String(), Required: false, Default: "CURRENT_TIMESTAMP", Index: 2},
		{Name: models.ENDKEY, Type: models.TIMESTAMP.String(), Required: false, Index: 3},
		{Name: RootID(DBSchema.Name), Type: models.INTEGER.String(), ForeignTable: DBSchema.Name, Required: true, Index: 4, Label: "template delegated"},
		{Name: RootID("dest_table"), Type: models.INTEGER.String(), Required: true, Readonly: true, Label: "reference", Index: 5},
		{Name: "read_access", Type: models.BOOLEAN.String(), Required: false, Default: true, Index: 6, Label: "read access"},
		{Name: "update_access", Type: models.BOOLEAN.String(), Required: false, Default: true, Index: 7, Label: "update access"},
		{Name: "delete_access", Type: models.BOOLEAN.String(), Required: false, Default: true, Index: 9, Label: "delete access"},
		{Name: RootID(DBDelegation.Name), Type: models.INTEGER.String(), Required: false, ForeignTable: DBDelegation.Name, Index: 10},
	},
} // TODO PERMISSION

var OWNPERMISSIONEXCEPTION = []string{DBNotification.Name, DBDelegation.Name, DBDashboard.Name, DBDashboardElement.Name, DBDashboardMathField.Name}
var AllPERMISSIONEXCEPTION = []string{DBNotification.Name, DBViewAttribution.Name, DBUser.Name, DBFilter.Name, DBFilterField.Name, DBComment.Name}
var POSTPERMISSIONEXCEPTION = []string{DBEmailSended.Name, DBEmailSendedUser.Name, DBRequest.Name, DBConsentResponse.Name, DBDelegation.Name, DBShare.Name}
var PUPERMISSIONEXCEPTION = []string{DBTask.Name, DBEmailResponse.Name}
var PERMISSIONEXCEPTION = []string{
	DBDashboard.Name, DBView.Name, DBTask.Name, DBShare.Name,
	DBDelegation.Name, DBRequest.Name, DBWorkflow.Name,
	DBEntity.Name, DBSchema.Name,
	DBSchemaField.Name, DBComment.Name, DBDataAccess.Name,
} // override permission checkup

var ROOTTABLES = []models.SchemaModel{DBSchemaField, DBUser, DBWorkflow, DBView, DBRequest, DBSchema, DBPermission, DBFilter, DBFilterField, DBEntity,
	DBRole, DBDataAccess, DBNotification, DBEntityUser, DBRoleAttribution, DBShare,
	DBConsent, DBTask, DBWorkflowSchema, DBRolePermission, DBHierarchy, DBViewAttribution,
	DBDashboard, // DBDashboardElement, DBDashboardMathField, DBDashboardLabel,
	DBComment, DBDelegation,
	DBConsentResponse, DBEmailTemplate,
	DBTrigger, DBTriggerRule, DBTriggerCondition, DBTriggerDestination,
	DBFieldAutoFill, DBFieldRule, DBFieldCondition,
	DBViewSchema,
	DBEmailSended, DBEmailTemplate, DBEmailResponse, DBEmailSendedUser, DBEmailList,
}

var NOAUTOLOADROOTTABLES = []models.SchemaModel{DBSchema, DBSchemaField, DBPermission, DBView, DBWorkflow}
var NOAUTOLOADROOTTABLESSTR = []string{DBSchema.Name, DBSchemaField.Name, DBPermission.Name, DBView.Name, DBWorkflow.Name}

func IsRootDB(name string) bool {
	if len(name) > 1 {
		return strings.Contains(name[:2], "db")
	} else {
		return false
	}
}
func RootID(name string) string {
	if IsRootDB(name) {
		return name + "_id"
	} else {
		return RootName(name) + "_id"
	}
}

func RootName(name string) string { return "db" + name }

var FieldRuleDBField = RootID(DBFieldRule.Name)
var ConsentDBField = RootID(DBConsent.Name)
var SchemaDBField = RootID(DBSchema.Name)
var SchemaFieldDBField = RootID(DBSchemaField.Name)
var RequestDBField = RootID(DBRequest.Name)
var TaskDBField = RootID(DBTask.Name)
var NotificationDBField = RootID(DBNotification.Name)
var DataAccessDBField = RootID(DBDataAccess.Name)
var WorkflowDBField = RootID(DBWorkflow.Name)
var WorkflowSchemaDBField = RootID(DBWorkflowSchema.Name)
var UserDBField = RootID(DBUser.Name)
var EntityDBField = RootID(DBEntity.Name)
var DestTableDBField = RootID("dest_table")
var FilterDBField = RootID(DBFilter.Name)
var FilterFieldDBField = RootID(DBFilterField.Name)
var ViewFilterDBField = "view_" + RootID(DBFilter.Name)
var ViewDBField = RootID(DBView.Name)
var DashboardDBField = RootID(DBDashboard.Name)
var DashboardMathDBField = RootID(DBDashboardMathField.Name)
var DashboardElementDBField = RootID(DBDashboardElement.Name)
var ViewAttributionDBField = RootID(DBViewAttribution.Name)
var TriggerDBField = RootID(DBTrigger.Name)
var EmailTemplateDBField = RootID(DBEmailTemplate.Name)
var EmailSendedDBField = RootID(DBEmailSended.Name)
var TriggerRuleDBField = RootID(DBTriggerRule.Name)
var DelegationDBField = RootID(DBDelegation.Name)
