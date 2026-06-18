INSERT INTO "dbschema_column" ("id", "active", "is_draft", "name", "label", "type", "description", "placeholder", "default_value", "index", "readonly", "required", "read_level", "dbschema_id", "constraints", "link_id", "hidden", "translatable", "transform_function", "group_by", "in_resume", "subsection", "info") VALUES
(6044367,	't',	'f',	'volume',	'volume du journal',	'varchar(128)',	NULL,	NULL,	NULL,	13,	'f',	'f',	'normal',	46,	NULL,	NULL,	'f',	't',	NULL,	NULL,	NULL,	NULL,	NULL),
(6044368,	't',	'f',	'pages',	'pages du journal',	'varchar(128)',	NULL,	NULL,	NULL,	14,	'f',	'f',	'normal',	46,	NULL,	NULL,	'f',	't',	NULL,	NULL,	NULL,	NULL,	NULL);

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(152,	't',	'f',	6044367,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	97,	'f',	'f',	NULL,	NULL,	NULL),
(153,	't',	'f',	6044368,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	97,	'f',	'f',	NULL,	NULL,	NULL);

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(155,	't',	'f',	6044367,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	354,	'f',	'f',	NULL,	NULL,	NULL),
(156,	't',	'f',	6044368,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	354,	'f',	'f',	NULL,	NULL,	NULL);

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(157,	't',	'f',	6044367,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	965,	'f',	'f',	NULL,	NULL,	NULL),
(158,	't',	'f',	6044368,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	965,	'f',	'f',	NULL,	NULL,	NULL);


# INSERT NEW FIELD

INSERT INTO "dbschema_column" ("id", "active", "is_draft", "name", "label", "type", "description", "placeholder", "default_value", "index", "readonly", "required", "read_level", "dbschema_id", "constraints", "link_id", "hidden", "translatable", "transform_function", "group_by", "in_resume", "subsection", "info") VALUES
(6044342,	't',	'f',	'abstract_publication',	'abstract finalisée',	'upload',	NULL,	NULL,	NULL,	-15,	'f',	'f',	'normal',	44,	NULL,	NULL,	'f',	't',	NULL,	NULL,	NULL,	'acte de publication',	NULL),
(16846,	't',	'f',	'abstract_publication',	'abstract finalisée',	'upload',	NULL,	NULL,	NULL,	-15,	'f',	'f',	'normal',	43,	NULL,	NULL,	'f',	't',	NULL,	NULL,	NULL,	'acte de publication',	NULL),
(864,	't',	'f',	'abstract_publication',	'abstract finalisée',	'upload',	NULL,	NULL,	NULL,	-15,	'f',	'f',	'normal',	42,	NULL,	NULL,	'f',	't',	NULL,	NULL,	NULL,	'acte de publication',	NULL);


# INSERT FILTERS

INSERT INTO "dbfilter" ("id", "active", "is_draft", "name", "is_view", "is_selected", "dbschema_id", "dbuser_id", "dbentity_id", "elder", "dashboard_restricted", "hidden") VALUES
(103,	't',	'f',	'post filter abs poster with abstract',	't',	'f',	42,	NULL,	NULL,	'all',	'f',	'f');


INSERT INTO "dbfilter" ("id", "active", "is_draft", "name", "is_view", "is_selected", "dbschema_id", "dbuser_id", "dbentity_id", "elder", "dashboard_restricted", "hidden") VALUES
(100,	't',	'f',	'post filter abs presentations with congress proceedings',	't',	'f',	44,	NULL,	NULL,	'all',	'f',	'f'),
(101,	't',	'f',	'post filter abs presentations without acts',	't',	'f',	43,	NULL,	NULL,	'all',	'f',	'f');


# INSERT FILTER FIELDS FOR CONFERENCE

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(133,	't',	'f',	692,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(134,	't',	'f',	311,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(135,	't',	'f',	304,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(136,	't',	'f',	692,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(137,	't',	'f',	305,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(138,	't',	'f',	308,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(139,	't',	'f',	312,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(140,	't',	'f',	314,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(141,	't',	'f',	307,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(142,	't',	'f',	306,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(143,	't',	'f',	316,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(144,	't',	'f',	303,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(145,	't',	'f',	317,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(146,	't',	'f',	427,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(147,	't',	'f',	544,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(148,	't',	'f',	318,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL),
(149,	't',	'f',	864,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	103,	'f',	'f',	NULL,	NULL,	NULL);

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(98,	't',	'f',	2501,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(99,	't',	'f',	692,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(100,	't',	'f',	311,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(101,	't',	'f',	304,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(102,	't',	'f',	692,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(103,	't',	'f',	7521,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(104,	't',	'f',	305,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(105,	't',	'f',	308,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(106,	't',	'f',	312,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(107,	't',	'f',	314,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(108,	't',	'f',	307,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(109,	't',	'f',	306,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(110,	't',	'f',	316,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(111,	't',	'f',	303,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(112,	't',	'f',	317,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(113,	't',	'f',	427,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(114,	't',	'f',	544,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(115,	't',	'f',	318,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL),
(150,	't',	'f',	864,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	424,	'f',	'f',	NULL,	NULL,	NULL);

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(116,	't',	'f',	2501,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(117,	't',	'f',	311,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(118,	't',	'f',	7521,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(119,	't',	'f',	305,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(120,	't',	'f',	308,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(121,	't',	'f',	312,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(122,	't',	'f',	314,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(123,	't',	'f',	307,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(124,	't',	'f',	306,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(125,	't',	'f',	316,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(126,	't',	'f',	303,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(127,	't',	'f',	317,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(128,	't',	'f',	427,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(129,	't',	'f',	544,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(130,	't',	'f',	318,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(131,	't',	'f',	41215,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	't',	NULL,	NULL,	NULL),
(132,	't',	'f',	304,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL),
(151,	't',	'f',	864,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	49632,	'f',	'f',	NULL,	NULL,	NULL);


INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(2,	't',	'f',	581,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(3,	't',	'f',	345,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(4,	't',	'f',	383,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(5,	't',	'f',	582,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(6,	't',	'f',	346,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(7,	't',	'f',	334,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(8,	't',	'f',	347,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(9,	't',	'f',	335,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(10,	't',	'f',	343,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(11,	't',	'f',	337,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(12,	't',	'f',	332,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(13,	't',	'f',	336,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(14,	't',	'f',	344,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(15,	't',	'f',	673,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(16,	't',	'f',	694,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(17,	't',	'f',	674,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(19,	't',	'f',	333,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(20,	't',	'f',	675,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(21,	't',	'f',	676,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(22,	't',	'f',	677,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(23,	't',	'f',	340,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	'f',	NULL,	NULL,	NULL),
(24,	't',	'f',	6044342,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	100,	'f',	't',	NULL,	NULL,	NULL);

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(79,	't',	'f',	6044342,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	637,	'f',	'f',	NULL,	NULL,	NULL);


INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(78,	't',	'f',	6044342,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	986,	'f',	'f',	NULL,	NULL,	NULL);


# INSERT FILTER FIELDS FOR PRESENTATION

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(82,	't',	'f',	320,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(83,	't',	'f',	562,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(84,	't',	'f',	330,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(85,	't',	'f',	328,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(86,	't',	'f',	324,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(87,	't',	'f',	321,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(88,	't',	'f',	331,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(89,	't',	'f',	319,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(90,	't',	'f',	438,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(91,	't',	'f',	322,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(92,	't',	'f',	323,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(93,	't',	'f',	693,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(94,	't',	'f',	327,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	'f',	NULL,	NULL,	NULL),
(95,	't',	'f',	16846,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	101,	'f',	't',	NULL,	NULL,	NULL);

INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(96,	't',	'f',	16846,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	453,	'f',	'f',	NULL,	NULL,	NULL);


INSERT INTO "dbfilter_field" ("id", "active", "is_draft", "dbschema_column_id", "value", "operator", "separator", "dir", "index", "width", "is_own", "dbfilter_id", "is_task_concerned", "force_not_readonly", "name", "is_hierarch_concerned", "is_hierarch_only") VALUES
(97,	't',	'f',	16846,	NULL,	NULL,	NULL,	NULL,	1,	NULL,	'f',	555,	'f',	'f',	NULL,	NULL,	NULL);


INSERT INTO "dbworkflow" ("id", "active", "is_draft", "name", "description", "is_meta", "dbschema_id", "view_dbfilter_id") VALUES
(36,	't',	'f',	'Créer un poster avec abstract',	'new posters with abstract workflow',	'f',	42,	92);



# DELETE workflow before

DELETE * FROM "dbworkflow_schema" WHERE dbworkflow_id IN (34,35);

# IMPORT CONFERENCE

INSERT INTO "dbworkflow_schema" ("id", "active", "is_draft", "name", "description", "index", "urgency", "priority", "optionnal", "hub", "dbworkflow_id", "dbschema_id", "dbuser_id", "dbentity_id", "wrapped_dbworkflow_id", "before_hierarchical_validation", "custom_progressing_status", "view_dbfilter_id", "readonly_not_assignee", "assign_to_creator", "override_state_completed", "override_state_dismiss", "override_state_refused") VALUES
(9,	't',	'f',	'validation de l''abstract',	NULL,	1,	'normal',	'normal',	'f',	'f',	35,	44,	NULL,	NULL,	NULL,	NULL,	NULL,	100,	't',	't',	'valider l''abstract',	'revenir à l''étape précédente',	'abandonner la publication'),
(10,	't',	'f',	'publication primée ? ',	NULL,	3,	'normal',	'normal',	't',	'f',	35,	44,	NULL,	NULL,	NULL,	NULL,	NULL,	986,	't',	't',	'valider la publication',	'revenir à l''étape précédente',	'abandonner la publication'),
(11,	't',	'f',	'publication en cours de validation par la conférence',	NULL,	2,	'normal',	'normal',	't',	'f',	35,	44,	NULL,	NULL,	NULL,	NULL,	NULL,	637,	't',	't',	'valider la publication',	'revenir à l''étape précédente',	'abandonner la publication');


INSERT INTO "dbworkflow_schema" ("id", "active", "is_draft", "name", "description", "index", "urgency", "priority", "optionnal", "hub", "dbworkflow_id", "dbschema_id", "dbuser_id", "dbentity_id", "wrapped_dbworkflow_id", "before_hierarchical_validation", "custom_progressing_status", "view_dbfilter_id", "readonly_not_assignee", "assign_to_creator", "override_state_completed", "override_state_dismiss", "override_state_refused") VALUES
(18,	't',	'f',	'publication en cours de validation et d''acceptation de la conférence',	NULL,	2,	'normal',	'normal',	'f',	'f',	36,	42,	NULL,	NULL,	NULL,	NULL,	NULL,	424,	't',	't',	'valider la publication',	'revenir à l''étape précédente',	'abandonner la publication'),
(19,	't',	'f',	'publication primée ? ',	NULL,	3,	'normal',	'normal',	't',	'f',	36,	42,	NULL,	NULL,	NULL,	NULL,	NULL,	49632,	't',	't',	'valider la publication',	'NA',	'abandonner la publication'),
(20,	't',	'f',	'validation de l''abstract',	NULL,	1,	'normal',	'normal',	'f',	'f',	36,	42,	NULL,	NULL,	NULL,	NULL,	NULL,	103,	't',	't',	'valider l''abstract',	'revenir à l''étape précédente',	'abandonner la publication');

# IMPORT PRESENTATION

INSERT INTO "dbworkflow_schema" ("id", "active", "is_draft", "name", "description", "index", "urgency", "priority", "optionnal", "hub", "dbworkflow_id", "dbschema_id", "dbuser_id", "dbentity_id", "wrapped_dbworkflow_id", "before_hierarchical_validation", "custom_progressing_status", "view_dbfilter_id", "readonly_not_assignee", "assign_to_creator", "override_state_completed", "override_state_dismiss", "override_state_refused") VALUES
(15,	't',	'f',	'publication primée ? ',	NULL,	3,	'normal',	'normal',	't',	'f',	34,	43,	NULL,	NULL,	NULL,	NULL,	NULL,	555,	't',	't',	'valider la publication',	'revenir à l''étape précédente',	'abandonner la publication'),
(5,	't',	'f',	'publication en cours de validation par la conférence',	NULL,	2,	'normal',	'normal',	't',	'f',	34,	43,	NULL,	NULL,	NULL,	NULL,	NULL,	453,	't',	't',	'valider la publication',	'revenir à l''étape précédente',	'abandonner la publication'),
(100,	't',	'f',	'validation de l''abstract',	NULL,	1,	'normal',	'normal',	'f',	'f',	34,	43,	NULL,	NULL,	NULL,	NULL,	NULL,	101,	't',	't',	'valider l''abstract',	'revenir à l''étape précédente',	'abandonner la publication');


DELETE * FROM dbtriggers WHERE mode = 'manual' and dbschema_id IN (44,43)

INSERT INTO "dbtriggers" ("id", "active", "is_draft", "name", "type", "mode", "dbschema_id", "on_write", "on_update", "description", "job_duration", "job_start_date", "on_update_step") VALUES
(2,	't',	'f',	'envoyer un email : "autorisation de publication d''une conférence"',	'mail',	'manual',	44,	'f',	't',	'Nous te proposons d''envoyer cet email aux personne en charge de valider que les informations contenues dans cette publication ne sont pas confidentielles. Cet email n''est cependant pas obligatoire',	NULL,	NULL,	5),
(3,	't',	'f',	'envoyer un email : "autorisation de publication d''une présentation sans relecture"',	'mail',	'manual',	43,	'f',	't',	'Nous te proposons d''envoyer cet email aux personne en charge de valider que les informations contenues dans cette publication ne sont pas confidentielles. Cet email n''est cependant pas obligatoire',	NULL,	NULL,	NULL);