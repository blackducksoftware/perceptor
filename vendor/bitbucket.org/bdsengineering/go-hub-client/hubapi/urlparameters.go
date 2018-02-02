package hubapi

import (
	"fmt"
	"net/url"
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
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := []string{}
	for _, key := range keys {
		val := dict[key]
		pairs = append(pairs, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(val)))
	}
	return strings.Join(pairs, "&")
}
