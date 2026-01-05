package utils

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func ToTime(value interface{}) time.Time {
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

func ToInt(value interface{}) int {
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

func ToInt32(value interface{}) int32 {
	switch v := value.(type) {
	case int32:
		return v
	case int:
		return int32(v)
	case int64:
		return int32(v)
	case float64:
		return int32(v)
	default:
		return 0
	}
}

func Contains[T int | string](data []T, value T) bool {
	for _, v := range data {
		if v == value {
			return true
		}
	}
	return false
}
