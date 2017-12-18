package hubclient

import (
	"bitbucket.org/bdsengineering/go-hub-client/hubapi"
	log "github.com/sirupsen/logrus"
)

func (c *Client) GetExternalExtension(link hubapi.ResourceLink) (*hubapi.ExternalExtension, error) {

	var extension hubapi.ExternalExtension
	err := c.httpGetJSON(link.Href, &extension, 200)

	if err != nil {
		log.Errorf("Error trying to retrieve an external extension: %+v.", err)
		return nil, err
	}

	return &extension, nil
}

func (c *Client) UpdateExternalExtension(extension *hubapi.ExternalExtension) error {

	err := c.httpPutJSON(extension.Meta.Href, &extension, hubapi.ContentTypeExtensionJSON, 200)

	if err != nil {
		log.Errorf("Error trying to update an external extension: %+v.", err)
		return err
	}

	return nil
}
