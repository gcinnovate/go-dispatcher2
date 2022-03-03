package models

import (
	"fmt"
	"log"
	"time"

	"github.com/gcinnovate/go-dispatcher2/db"
)

func init() {
	rows, err := db.GetDB().Queryx("SELECT * FROM servers")
	defer rows.Close()

	if err != nil {
		log.Fatalf("Failed to load servers", err)
	}
	for rows.Next() {
		srv := &Server{}
		s := &srv.s
		err := rows.StructScan(&s)
		if err != nil {
			log.Fatalln("Server Loading ==>", err)
		}
		// fmt.Printf("=>>>>>>%#v", s)
		ServerMap = make(map[string]Server)
		ServerMap[s.UID] = *srv

	}
}

// ServerMap is the List of Servers
var ServerMap map[string]Server

// ServerID is the id for the server
type ServerID int64

// Server is our user object
type Server struct {
	s struct {
		ID                      ServerID  `db:"id" 						json:"id"`
		UID                     string    `db:"uid" 					json:"uid"`
		Name                    string    `db:"name" 					json:"name"`
		Username                string    `db:"username"				json:"username"`
		Password                string    `db:"password"				json:"password"`
		IsProxyServer           string    `db:"is_proxy_server"			json:"is_proxy_server"` // whether request referencing url receives response as is
		SystemType              string    `db:"system_type"				json:"system_type"`        // the type of system e.g DHIS2, Other is the default
		EndPointType            string    `db:"endpoint_type"			json:"endpoint_type"`     // e.g /dataValueSets,
		AuthToken               string    `db:"auth_token"				json:"auth_token"`
		IPAddress               string    `db:"ipaddress"				json:"ipaddress"` // Usefull for setting Trusted Proxies
		URL                     string    `db:"url"						json:"url"`
		CCURLS                  []string  `db:"cc_url"					json:"cc_url"`                 // just an additional URL to receive same request
		CallbackURL             string    `db:"callback_url" 			json:"callback_url"`      // receives response on success call to url
		HTTPMethod              string    `db:"http_method"				json:"http_method"`        // the HTTP Method used when calling the url
		AuthMethod              string    `db:"auth_method"				json:"auth_method"`        // the Authentication Method used
		AllowCallbacks          bool      `db:"allow_callbacks"			json:"allow_callbacks"` // Whether to allow calling sending callbacks
		AllowCopies             bool      `db:"allow_copies"			json:"allow_copies"`       // Whether to allow copying similar request to CCURLs
		UseSSL                  bool      `db:"use_ssl" 				json:"use_ssl"`
		ParseResponses          bool      `db:"parse_responses" 		json:"parse_responses"`
		SSLClientCertKeyFile    string    `db:"ssl_client_certkey_file"	json:"ssl_client_certkey_file"`
		StartOfSubmissionPeriod string    `db:"start_submission_period"	json:"start_submission_period"`
		EndOfSubmissionPeriod   string    `db:"end_submission_period"	json:"end_submission_period"`
		XMLResponseXPATH        string    `db:"xml_response_xpath" 		json:"xml_response_xpath"`
		JSONResponseXPATH       string    `db:"json_response_xpath" 	json:"json_response_xpath"`
		Suspended               bool      `db:"suspended" 				json:"suspended"`
		Created                 time.Time `db:"created" 				json:"created_at"`
		Updated                 time.Time `db:"updated" 				json:"updated_at"`
	}
}

// ServerAllowedApps hold servers and servers they allow to communicate with
type ServerAllowedApps struct {
	ID             int64      `db:"id" 				json:"id"`
	ServerID       ServerID   `db:"server_id" 		json:"server_id"`
	AllowedServers []ServerID `db:"allowed_servers" json:"allowed_servers"`
}

// ID return the id of this request
func (s *Server) ID() ServerID { return s.s.ID }

// UID returns the uid of the server/app
func (s *Server) UID() string { return s.s.UID }

// SystemType return the type of system/app it is
func (s *Server) SystemType() string { return s.s.SystemType }

// AuthToken return the Authentication token for this server
func (s *Server) AuthToken() string { return s.s.AuthToken }

// URL returns the URL for the server
func (s *Server) URL() string { return s.s.URL }

// HTTPMethod returns the method used when calling the URL
func (s *Server) HTTPMethod() string { return s.s.HTTPMethod }

// AllowCallbacks returns whether server allows callbacks
func (s *Server) AllowCallbacks() bool { return s.s.AllowCallbacks }

// CallbackURL return the server callback url
func (s *Server) CallbackURL() string { return s.s.CallbackURL }

// ParseResponses return whether we shold parse the server's responses
func (s *Server) ParseResponses() bool { return s.s.ParseResponses }

// EndOfSubmissionPeriod returns the end of the submission period for the server
func (s *Server) EndOfSubmissionPeriod() string { return s.s.EndOfSubmissionPeriod }

// StartOfSubmissionPeriod returns the start of the submission period for the server
func (s *Server) StartOfSubmissionPeriod() string { return s.s.StartOfSubmissionPeriod }

// Suspended returns whether the server is suspended
func (s *Server) Suspended() ServerID { return s.s.ID }

// CreatedOn return time when Server/App was created
func (s *Server) CreatedOn() time.Time { return s.s.Created }

// UpdatedOn return time when server/app was updated
func (s *Server) UpdatedOn() time.Time { return s.s.Updated }

// GetServerByID returns server object using id
func GetServerByID(id int64) Server {
	srv := Server{}
	err := db.GetDB().Get(&srv.s, "SELECT * FROM servers WHERE id = $1", id)

	if err != nil {
		fmt.Printf("Error geting server: [%v]", err)
		return Server{}
	}
	return srv

}
