package hubapi

import (
	"fmt"
	"sort"
	"strings"
)

// URLParameters describes types used as parameter models
// for GET endpoints.
type URLParameters interface {
	Parameters() map[string]string
}

// ParameterString takes a URLParameters object
// and converts it to a string which can be added to
// a URL.
func ParameterString(params URLParameters) string {
	dict := params.Parameters()

	var keys []string
	for k, _ := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := []string{}
	for _, key := range keys {
		val := dict[key]
		// TODO there should be some real URL encoding eventually
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, val))
	}
	return strings.Join(pairs, "&")
}
