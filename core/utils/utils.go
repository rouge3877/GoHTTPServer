package utils

import "net/url"

func ParseQuery(rawQuery string) map[string]string {
	result := make(map[string]string)
	values, _ := url.ParseQuery(rawQuery)
	for k, v := range values {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}
