package utils

// StringsContain
func StringsContain(strs []string, str string) bool {
	for i := range strs {
		if strs[i] == str {
			return true
		}
	}
	return false
}

type KeyValueGetter func(key string) (interface{}, bool)
type KeyValueIterator func(KeyValueIterateFunc)
type KeyValueIterateFunc func(key, val string) (stop bool)

const (
	LogKeyDagInsID = "dagInsId"
)
