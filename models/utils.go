package models

import (
	"errors"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"strings"
)

func GetDHIS2BaseURL(url string) (string, error) {
	if strings.Contains(url, "/api/") {
		pos := strings.Index(url, "/api/")
		return url[:pos], nil
	}
	return url, errors.New("URL doesn't contain /api/ part")
}

type Client struct {
	RestClient *resty.Client
	BaseURL    string
	// AuthToken  string
}

func (s *Server) NewClient() (*Client, error) {
	client := resty.New()
	baseUrl, err := GetDHIS2BaseURL(s.URL())
	if err != nil {
		log.WithFields(log.Fields{
			"URL": s.URL(), "Error": err}).Error("Failed to get base URL from URL")
		return nil, err
	}
	client.SetBaseURL(baseUrl + "/api")
	client.SetHeaders(map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	switch s.AuthMethod() {
	case "Basic":
		client.SetBasicAuth(s.Username(), s.Password())
	case "Token":
		client.SetAuthScheme("Token")
		client.SetAuthToken(s.AuthToken())
	}
	return &Client{
		RestClient: client,
		BaseURL:    baseUrl + "/api",
		// AuthToken:  s.AuthToken(),
	}, nil
}

func (c *Client) GetResource(resourcePath string, params map[string]string) (*resty.Response, error) {
	request := c.RestClient.R()

	if params != nil {
		request.SetQueryParams(params)
	}

	resp, err := request.Get(resourcePath)
	if err != nil {
		log.Fatalf("Error when calling `GetResource`: %v", err)
	}
	return resp, err
}

func (c *Client) PostResource(resourcePath string, data interface{}) (*resty.Response, error) {
	resp, err := c.RestClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Post(resourcePath)
	if err != nil {
		log.Fatalf("Error when calling `PostResource`: %v", err)
	}
	return resp, err
}

func (c *Client) PutResource(resourcePath string, data interface{}) (*resty.Response, error) {
	resp, err := c.RestClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Put(resourcePath)
	if err != nil {
		log.Fatalf("Error when calling `PutResource`: %v", err)
	}
	return resp, err
}

func (c *Client) DeleteResource(resourcePath string) (*resty.Response, error) {
	resp, err := c.RestClient.R().
		Delete(resourcePath)
	if err != nil {
		log.Fatalf("Error when calling `DeleteResource`: %v", err)
	}
	return resp, err
}

func (c *Client) PatchResource(resourcePath string, data interface{}) (*resty.Response, error) {
	resp, err := c.RestClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(data).
		Patch(resourcePath)
	if err != nil {
		log.Fatalf("Error when calling `PatchResource`: %v", err)
	}
	return resp, err
}
