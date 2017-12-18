package hubapi

import (
	"fmt"
)

type Meta struct {
	Allow []string       `json:"allow"`
	Href  string         `json:"href"`
	Links []ResourceLink `json:"links"`
}

type ResourceLink struct {
	Rel   string `json:"rel"`
	Href  string `json:"href"`
	Label string `json:"label"`
	Name  string `json:"name"`
}

func (m *Meta) FindLinkByRel(rel string) (*ResourceLink, error) {

	for _, l := range m.Links {
		if l.Rel == rel {
			return &l, nil
		}
	}

	return nil, fmt.Errorf("no relation '%s' found", rel)
}
