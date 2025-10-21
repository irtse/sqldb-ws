package triggers

import (
	"fmt"
	ds "sqldb-ws/domain/schema/database_resources"
	sm "sqldb-ws/domain/schema/models"
	"sqldb-ws/domain/utils"
	"sync"
	"time"
)

var layouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02",
	"01/02/2006",
	"02/01/2006",
	"2006-01-02T15:04:05",
	"January 2, 2006 3:04 PM",
}

func parseDate(input string) (time.Time, error) {
	for _, layout := range layouts {
		if t, err := time.Parse(layout, input); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format: %s", input)
}

var cacheTriggerMutex sync.Mutex
var cacheTriggerIDS = map[time.Time]string{}

func ShouldExecLater(trigger map[string]interface{}) bool {
	if trigger["job_start_date"] != nil {
		if t, err := parseDate(utils.GetString(trigger, "job_start_date")); err == nil {
			if time.Now().UTC().After(t.UTC()) {
				return false
			} else {
				cacheTriggerMutex.Lock()
				cacheTriggerIDS[t] = utils.GetString(trigger, utils.SpecialIDParam)
				cacheTriggerMutex.Unlock()
				return true
			}
		}
	}
	return false
}

func ShouldExecJob(trigger map[string]interface{}) {
	if trigger["job_duration"] != nil {
		t := time.Now().UTC().Add(time.Duration(utils.GetInt(trigger, "job_duration")) * time.Second)
		cacheTriggerMutex.Lock()
		cacheTriggerIDS[t] = utils.GetString(trigger, utils.SpecialIDParam)
		cacheTriggerMutex.Unlock()
	}
}

func Exec(fromSchema *sm.SchemaModel, record utils.Record, domain utils.DomainITF) {
	for true {
		for k, id := range cacheTriggerIDS {
			utc := k.UTC()
			if time.Now().UTC().Before(utc) {
				if res, err := domain.GetDb().ClearQueryFilter().SelectQueryWithRestriction(ds.DBTrigger.Name, map[string]interface{}{
					utils.SpecialIDParam: id,
				}, false); err == nil && len(res) > 0 {
					NewTrigger(domain).ExecTrigger(fromSchema, record, res[0])
					cacheTriggerMutex.Lock()
					delete(cacheTriggerIDS, k)
					cacheTriggerMutex.Unlock()
				}
			}
		}
		time.Sleep(time.Minute * 1)
	}
}
