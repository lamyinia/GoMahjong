package utils

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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

func ToString(value interface{}) string {
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

func ToIntArray(value interface{}) [4]int {
	var result [4]int
	switch v := value.(type) {
	case []interface{}:
		for i := 0; i < 4 && i < len(v); i++ {
			result[i] = ToInt(v[i])
		}
	case []int:
		for i := 0; i < 4 && i < len(v); i++ {
			result[i] = v[i]
		}
	case [4]int:
		return v
	}
	return result
}

func ToStringArray(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = ToString(item)
		}
		return result
	case []string:
		return v
	default:
		return []string{}
	}
}
