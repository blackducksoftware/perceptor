package hubapi

import (
	"testing"
)

func TestGetListOptionsURLSerialization(t *testing.T) {
	limit := 3
	offset := 12
	q := "a?bc"
	gpo := GetListOptions{
		Limit:  &limit,
		Offset: &offset,
		// skip "Sort", meaning it will be nil, and not show up in the query string
		Q: &q,
	}
	actual := ParameterString(&gpo)
	expected := "limit=3&offset=12&q=a%3Fbc"
	if actual != expected {
		t.Errorf("URL parameters serialized incorrectly -- expected %s, got %s", expected, actual)
		t.Fail()
	}
}
