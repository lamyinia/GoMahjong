package persistence

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func toTime(value interface{}) time.Time {
	switch v := value.(type) {
	case primitive.DateTime:
		return v.Time()
	case primitive.Timestamp:
		return time.Unix(int64(v.T), 0)
	case time.Time:
		return v
	case *time.Time:
		if v != nil {
			return *v
		}
	default:
	}
	return time.Time{}
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case *string:
		if v != nil {
			return *v
		}
	default:
		return ""
	}
	return ""
}
