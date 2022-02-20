package models

import (
	"time"
)

// Server is our user object
type Server struct {
	ID                      int64     `db:"id"`
	UUID                    string    `db:"uuid" json:"uuid"`
	Name                    string    `db:"name" json:"name"`
	Username                string    `db:"username"`
	Password                string    `db:"password"`
	IsProxyServer           string    `db:"is_proxy_server"` // whether request referencing request receives response as is
	SystemType              string    `db:"system_type"`     // the type of system e.g DHIS2, Other is the default
	EndPointType            string    `db:"endpoint_type"`   // e.g /dataValueSets,
	AuthToken               string    `db:"auth_token"`
	IPAddress               string    `db:"ipaddress"` // Usefull for setting Trusted Proxies
	URL                     string    `db:"url"`
	CCURLS                  []string  `db:"cc_url"`          // just an additional URL to receive same request could an array
	CallbackURL             string    `db:"callback_url"`    // receives response on success call to url
	HTTPMethod              string    `db:"http_method"`     // the HTTP Method used when calling the url
	AuthMethod              string    `db:"auth_method"`     // the Authentication Method used
	AllowCallbacks          bool      `db:"allow_callbacks"` // Whether to allow calling sending callbacks
	AllowCopies             bool      `db:"allow_copies"`    // Whether to allow copying similar request to CCURLs
	UseSSL                  bool      `db:"use_ssl" json:"use_ssl"`
	ParseResponses          bool      `db:"parse_responses" json:"parse_responses"`
	SSLClientCertKeyFile    string    `db:"ssl_client_certkey_file"`
	StartOfSubmissionPeriod string    `db:"start_submission_period"`
	EndOfSubmissionPeriod   string    `db:"end_submission_period"`
	XMLResponseXPATH        string    `db:"xml_response_xpath" json:"xml_response_xpath"`
	JSONResponseXPATH       string    `db:"json_response_xpath" json:"json_response_xpath"`
	Suspended               bool      `db:"suspended" json:"suspended"`
	Created                 time.Time `db:"created" json:"created_at"`
	Updated                 time.Time `db:"updated" json:"updated_at"`
}

type ServerAllowedApps struct {
}
