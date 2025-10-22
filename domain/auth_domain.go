package domain

import (
	"errors"
	"sqldb-ws/domain/schema"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	connector "sqldb-ws/infrastructure/connector/db"
	"strconv"
	"strings"
)

var IsMaintenance = false

func SetToken(superAdmin bool, user string, token interface{}) (utils.Results, error) {
	if IsMaintenance {
		return utils.Results{}, errors.New("actually in maintenance... can't proceed")
	}
	return Domain(superAdmin, user, nil).Call( // replace token by a nil
		utils.AllParams(ds.DBUser.Name).RootRaw(), utils.Record{"token": token}, utils.UPDATE, GetQueryFilter(user))
}

func IsLogged(superAdmin bool, user string, token string) (utils.Results, error) {
	if IsMaintenance {
		return utils.Results{}, errors.New("actually in maintenance... can't proceed")
	}
	domain := Domain(superAdmin, user, nil)

	response, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBUser.Name, map[string]interface{}{
		"name":  connector.Quote(strings.ToLower(user)),
		"email": connector.Quote(strings.ToLower(user)),
	}, true)
	if err != nil || len(response) == 0 {
		return nil, err
	}
	resp := response[0]
	notifs, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBNotification.Name,
		map[string]interface{}{
			ds.UserDBField: resp[utils.SpecialIDParam],
			ds.EntityDBField: domain.GetDb().ClearQueryFilter().BuildSelectQueryWithRestriction(
				ds.DBEntityUser.Name,
				map[string]interface{}{
					ds.UserDBField: resp[utils.SpecialIDParam],
				}, false, "id"),
		}, true)
	if err != nil {
		return nil, err
	}
	n := utils.Results{}
	for _, notif := range notifs {
		int, err := strconv.Atoi(utils.ToString(notif["link_id"]))
		if err != nil {
			continue
		}
		sch, err := schema.GetSchemaByID(int64(int))
		if err != nil {
			continue
		}
		nn := utils.Record{
			utils.SpecialIDParam: utils.GetString(notif, utils.SpecialIDParam),
			sm.NAMEKEY:           utils.GetString(notif, sm.NAMEKEY),
			"description":        utils.GetString(notif, "description"),
			"link_path":          "/" + utils.MAIN_PREFIX + "/" + ds.DBNotification.Name + "?" + utils.RootRowsParam + "=" + utils.GetString(notif, "id"),
			"data_ref":           "@" + utils.ToString(sch.ID) + ":" + utils.ToString(notif[utils.RootDestTableIDParam]),
		}
		n = append(n, nn)
	}
	resp["notifications"] = n
	resp["token"] = token
	return utils.Results{resp}, nil
}

func GetQueryFilter(user string) string {
	return connector.FormatSQLRestrictionWhereByMap("", map[string]interface{}{
		"name":  connector.Quote(user),
		"email": connector.Quote(user),
	}, true)
}
