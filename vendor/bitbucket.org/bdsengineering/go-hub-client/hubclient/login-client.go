package hubclient

import (
	"fmt"
	"net/url"

	log "github.com/sirupsen/logrus"
)

func (c *Client) Login(username string, password string) error {

	loginURL := fmt.Sprintf("%s/j_spring_security_check", c.baseURL)
	formValues := url.Values{
		"j_username": {username},
		"j_password": {password},
	}

	resp, err := c.httpClient.PostForm(loginURL, formValues)

	if err != nil {
		log.Errorf("Error trying to login via form login: %+v.", err)
		return err
	}

	if resp.StatusCode != 204 {
		log.Errorf("Login: Got a %d response instead of a 204.", resp.StatusCode)
		return fmt.Errorf("got a %d response instead of a 204", resp.StatusCode)
	}

	if csrf := resp.Header.Get(HeaderNameCsrfToken); csrf != "" {
		c.haveCsrfToken = true
		c.csrfToken = csrf
	}

	log.Debugln("Login: Successfully authenticated")

	return nil
}
