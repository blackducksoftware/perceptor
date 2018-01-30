package hubapi

import (
	"testing"
)

func TestGetProjectOptionsURLSerialization(t *testing.T) {
	limit := 3
	offset := 12
	q := "abc"
	gpo := GetProjectsOptions{
		Limit:  &limit,
		Offset: &offset,
		// skip "Sort",
		Q: &q,
	}
	actual := ParameterString(&gpo)
	expected := "limit=3&offset=12&q=abc"
	if actual != expected {
		t.Errorf("URL parameters serialized incorrectly -- expected %s, got %s", expected, actual)
		t.Fail()
	}
}
