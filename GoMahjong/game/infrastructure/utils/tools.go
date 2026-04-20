package utils

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ToTime(value interface{}) time.Time {
	switch v := value.(type) {
	case primitive.DateTime:
		return v.Time()
	case time.Time:
		return v
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
	}
	return 0
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
	}
	return 0
}

func ToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	}
	return ""
}

func ToIntArray(value interface{}) [4]int {
	var result [4]int
	switch v := value.(type) {
	case []interface{}:
		for i := 0; i < len(v) && i < 4; i++ {
			result[i] = ToInt(v[i])
		}
	}
	return result
}

func ToStringArray(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, x := range v {
			result[i] = ToString(x)
		}
		return result
	case []string:
		return v
	}
	return nil
}
