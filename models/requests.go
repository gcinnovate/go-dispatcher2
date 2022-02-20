package models

import "time"

// Request represents our requests queue in the database
type Request struct {
	ID                 int64     `db: "id"`
	Source             int       `db: "source"`
	Destination        int       `db: "destination"`
	ContentType        string    `db: "ctype"`
	Body               string    `db: "body"`
	Status             string    `db: "status"`
	StatusCode         string    `db: "statuscode"`
	Retries            int       `db: "retries"`
	Errors             string    `db: "errors"`
	InSubmissoinPeriod bool      `db: "in_submission_period"`
	FrequencyType      string    `db: "frequency_type" json:"frequency_type"`
	Period             string    `db: "period" json:"period"`
	Day                string    `db: "day"`
	Week               string    `db: "week"`
	Month              string    `db: "month"`
	Year               int       `db: "year"`
	MSISDN             string    `db: "msisdn"`
	RawMsg             string    `db: "raw_msg"`
	Facility           string    `db: "facility"`
	District           string    `db: "district"`
	ReportType         string    `db: "report_type" json:"report_type"` // type of report as in source system
	Extras             string    `db: "extras"`
	Suspended          bool      `db: "suspended"`           // whether request is suspended
	BodyIsQueryParams  bool      `db: "body_is_query_param"` // whether body is to be used a query parameters
	SubmissionID       int64     `db: "submissionid"`        // a reference ID is source system
	URLSurffix         string    `db: "url_suffix"`
	Created            time.Time `db: "created"`
	Updated            time.Time `db: "updated"`
}

// RequestResponse keeps responses to requests
type RequestResponse struct {
	RequestID int64
	Response  string
}
