package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gcinnovate/go-dispatcher2/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// RequestID is the id for our request
type RequestID int64

// RequestStatus is the status for each request
type RequestStatus string

// constants for the status
const (
	RequestStatusReady     = RequestStatus("ready")
	RequestStatusPending   = RequestStatus("pending")
	RequestStatusExpired   = RequestStatus("expired")
	RequestStatusCompleted = RequestStatus("completed")
	RequestStatusFailed    = RequestStatus("failed")
	RequestStatusError     = RequestStatus("error")
	RequestStatusIgnored   = RequestStatus("ignored")
)

// Request represents our requests queue in the database
type Request struct {
	r struct {
		ID                 RequestID     `db:"id" 						json:"id"`
		UID                string        `db:"uid" 					json:"uid"`
		Source             int           `db:"source" 					json:"source"`
		Destination        int           `db:"destination" 			json:"destination"`
		ContentType        string        `db:"ctype" 					json:"contentType"`
		Body               string        `db:"body" 					json:"body"`
		Response           string        `db:"response" 				json:"response,omitempty"`
		Status             RequestStatus `db:"status" 					json:"status"`
		StatusCode         string        `db:"statuscode" 				json:"statusCode"`
		Retries            int           `db:"retries" 				json:"retries"`
		Errors             string        `db:"errors" 					json:"errors"`
		InSubmissoinPeriod bool          `db:"in_submission_period" 	json:"inSubmissoinPeriod"`
		FrequencyType      string        `db:"frequency_type" 			json:"frequencyType"`
		Period             string        `db:"period" 					json:"period"`
		Day                string        `db:"day" 					json:"day"`
		Week               string        `db:"week" 					json:"week"`
		Month              string        `db:"month" 					json:"month"`
		Year               string        `db:"year" 					json:"year"`
		MSISDN             string        `db:"msisdn" 					json:"msisdn"`
		RawMsg             string        `db:"raw_msg" 				json:"rawMsg"`
		Facility           string        `db:"facility" 				json:"facility"`
		District           string        `db:"district" 				json:"district"`
		ReportType         string        `db:"report_type" 			json:"report_type"` // type of report as in source system
		Extras             string        `db:"extras" 					json:"extras"`
		Suspended          bool          `db:"suspended" 				json:"suspended"`                // whether request is suspended
		BodyIsQueryParams  bool          `db:"body_is_query_param" 	json:"bodyIsQueryParams"` // whether body is to be used a query parameters
		SubmissionID       string        `db:"submissionid" 			json:"submissionId"`           // a reference ID is source system
		URLSurffix         string        `db:"url_suffix" 				json:"urlSurffix"`
		Created            time.Time     `db:"created" 				json:"created"`
		Updated            time.Time     `db:"updated" 				json:"updated"`
		// OrgID              OrgID         `db:"org_id"          			json:"org_id"` // Lets add these later
	}
}

// ID return the id of this request
func (r *Request) ID() RequestID { return r.r.ID }

// Status returns the status of the request
func (r *Request) Status() RequestStatus { return r.r.Status }

// StatusCode reture the statuscode of the request
func (r *Request) StatusCode() string { return r.r.StatusCode }

// Period returns the period of the request
func (r *Request) Period() string { return r.r.Period }

// ContentType returns the contentType of the request
func (r *Request) ContentType() string { return r.r.ContentType }

// Errors return the errors after processing requests
func (r *Request) Errors() string { return r.r.Errors }

// BodyIsQueryParams returns whether request body is used as query params
func (r *Request) BodyIsQueryParams() bool { return r.r.BodyIsQueryParams }

// URLSurffix returns the url surffix used when submitting request
func (r *Request) URLSurffix() string { return r.r.URLSurffix }

// Source return id of source app
func (r *Request) Source() int { return r.r.Source }

// Destination return id of destination app
func (r *Request) Destination() int { return r.r.Destination }

// CreatedOn return time when request was created
func (r *Request) CreatedOn() time.Time { return r.r.Created }

// UpdatedOn return time when request was updated
func (r *Request) UpdatedOn() time.Time { return r.r.Updated }

// NewRequest creates new request and saves it in DB
func NewRequest(c *gin.Context, db *sqlx.DB) error {
	source := utils.GetServer(c.Query("source"))
	destination := utils.GetServer(c.Query("destination"))
	fmt.Printf("Source>: %v, Destination: %v", source, destination)

	req := &Request{}
	r := &req.r
	r.Source = source
	r.Destination = destination
	r.UID = utils.GetUID()
	r.ContentType = c.Request.Header.Get("Content-Type")
	r.SubmissionID = c.Query("msgid")
	r.Period = c.Query("period")
	r.Week = c.Query("week")
	r.Month = c.Query("month")
	r.Year = c.Query("year")
	r.MSISDN = c.Query("msisdn")
	r.Facility = c.Query("facility")
	r.RawMsg = c.Query("rawMsg")
	if c.Query("isQueryParams") == "true" {
		r.BodyIsQueryParams = true
	}
	r.ReportType = c.Query("reportType")
	r.Errors = c.Query("extras")
	r.District = c.Query("district")

	switch r.ContentType {
	case "application/json":
		var body map[string]interface{} // validate based on dest system endpoint
		if err := c.BindJSON(&body); err != nil {
			fmt.Printf("Error reading json body %v", err)
		}
		b, _ := json.Marshal(body)
		fmt.Println(string(b))
		r.Body = string(b)
	case "application/xml":
		// var xmlBody interface{}
		xmlBody, err := c.GetRawData()
		if err != nil {
			fmt.Printf("Error reading xml body %v", err)
		}
		r.Body = string(xmlBody)
	default:
		body, err := c.GetRawData()
		if err != nil {
			fmt.Printf("Error reading body %v", err)
		}
		r.Body = string(body)
	}

	_, err := db.NamedExec(insertRequestSQL, r)
	if err != nil {
		fmt.Printf("ERROR INSERTING REQUEST", err)
	}

	return nil
}

const insertRequestSQL = `
INSERT INTO 
requests (source, destination, ctype, body, body_is_query_param, period, week, month, year,
			raw_msg, msisdn, facility, district, report_type, extras, url_suffix,
			created, updated) 
	VALUES(:source, :destination, :ctype, :body, :body_is_query_param, :period,
			:week, :month, :year, :raw_msg, :msisdn, :facility, :district, :report_type,
			:extras, :url_suffix, now(), now())`
