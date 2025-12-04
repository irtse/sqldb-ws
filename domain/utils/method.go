package utils

import "strings"

type Method int64

const (
	UNKNOWN   Method = 0
	SELECT    Method = 1
	CREATE    Method = 2
	UPDATE    Method = 3
	DELETE    Method = 4
	COUNT     Method = 5
	AVG       Method = 6
	MIN       Method = 7
	MAX       Method = 8
	SUM       Method = 9
	IMPORT    Method = 10
	WEBSOCKET Method = 11
)

func Found(name string) Method {
	switch strings.ToLower(name) {
	case "read":
		return SELECT
	case "write":
		return CREATE
	case "update":
		return UPDATE
	case "delete":
		return DELETE
	case "count":
		return COUNT
	case "avg":
		return AVG
	case "min":
		return MIN
	case "max":
		return MAX
	case "sum":
		return SUM
	}
	return SELECT
}
func (s Method) String() string {
	switch s {
	case SELECT:
		return "read"
	case CREATE:
		return "write"
	case UPDATE:
		return "update"
	case DELETE:
		return "delete"
	case COUNT:
		return "count"
	case AVG:
		return "avg"
	case MIN:
		return "min"
	case MAX:
		return "max"
	case SUM:
		return "sum"
	}
	return "unknown"
}

func (s Method) IsMath() bool {
	switch s {
	case COUNT, AVG, MIN, MAX, SUM:
		return true
	}
	return false
}

func (s Method) Method() string {
	switch s {
	case SELECT:
		return "get"
	case CREATE:
		return "post"
	case UPDATE:
		return "put"
	case DELETE:
		return "delete"
	case COUNT:
		return "count"
	case AVG:
		return "avg"
	case MIN:
		return "min"
	case MAX:
		return "max"
	case SUM:
		return "sum"
	}
	return "unknown"
}

func (s Method) Calling() string {
	switch s {
	case SELECT:
		return "Get"
	case CREATE:
		return "Create"
	case UPDATE:
		return "Update"
	case DELETE:
		return "Delete"
	case COUNT:
		return "Math"
	case AVG:
		return "Math"
	case MIN:
		return "Math"
	case MAX:
		return "Math"
	case SUM:
		return "Math"
	}
	return "unknown"
}
