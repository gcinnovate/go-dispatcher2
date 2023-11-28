package models

import (
	"encoding/json"
	"fmt"
)

// DataValue is a single Data Value Object
type DataValue struct {
	DataElement         string `json:"dataElement"`
	CategoryOptionCombo string `json:"categoryoptioncombo,omitempty"`
	Value               string `json:"value"`
}

// DataValuesRequest is the format for sending data values - JSON
type DataValuesRequest struct {
	DataSet              string      `json:"dataset"`
	Completed            string      `json:"completed"`
	Period               string      `json:"period"`
	OrgUnit              string      `json:"orgUnit"`
	AttributeOptionCombo string      `json:"attributeoptioncomb,omitempty"`
	DataValues           []DataValue `json:"dataValues"`
}

// BulkDataValuesRequest is the format for sending bulk data values -JSON
type BulkDataValuesRequest struct {
	DataValues []struct {
		DataElement string `json:"dataElement"`
		Period      string `json:"period"`
		OrgUnit     string `json:"orgUnit"`
		Value       string `json:"value"`
	} `json:"dataValues"`
}

// Conflict is the object for the conflicts returned by DHIS 2 API
type Conflict struct {
	Object string `json:"object"`
	Value  string `json:"value"`
}

// DataValuesResponse represents the format of the DHIS 2 API response
type DataValuesResponse struct {
	b struct {
		Status        string                 `json:""`
		Description   string                 `json:"description"`
		ResponseType  string                 `json:"responseType"`
		ImportCount   ImportCount            `json:"importCount"`
		Conflicts     []Conflict             `json:"conflicts"`
		ImportOptions map[string]interface{} `json:"importOptions"`
	}
}

// ResponseStatus the status of a response
type ResponseStatus string

// ImportOptions the import options for dhis2 data import
type ImportOptions struct {
	IdSchemes                   map[string]string
	DryRun                      bool
	Async                       bool
	ImportStrategy              string
	MergeMode                   string
	ReportMode                  string
	SkipExistingCheck           bool
	Sharing                     bool
	SkipNotifications           bool
	SkipAudit                   bool
	DatasetAllowsPeriods        bool
	StrictPeriods               bool
	StrictDataElements          bool
	StrictCategoryOptionCombos  bool
	StrictAttributeOptionCombos bool
	StrictOrganisationUnits     bool
	RequireCategoryOptionCombo  bool
	RequireAttributeOptionCombo bool
	SkipPatternValidation       bool
	IgnoreEmptyCollection       bool
	Force                       bool
	FirstRowIsHeader            bool
	SkipLastUpdated             bool
	MergeDataValues             bool
	SkipCache                   bool
}

// ImportCount the import count in response
type ImportCount struct {
	Created  int
	Imported int
	Updated  int
	Ignored  int
	Deleted  int
	Total    int
}

type ConflictObject struct {
	Object    string
	Objects   map[string]string
	Value     string
	ErrorCode string
	Property  string
}

type Response struct {
	ResponseType    string
	Status          ResponseStatus
	ImportOptions   ImportOptions    `json:"importOptions,omitempty"`
	ImportCount     ImportCount      `json:"importCount,omitempty"`
	Stats           ImportCount      `json:"stats,omitempty"`
	Description     string           `json:"description,omitempty"`
	TypeReports     []any            `json:"typeReports,omitempty"`
	Conflicts       []ConflictObject `json:"conflicts,omitempty"`
	DataSetComplete string           `json:"dataSetComplete,omitempty"`
}

// ImportSummary for Aggregate and Async Requests
type ImportSummary struct {
	HTTPStatus     string `json:"httpStatus"`
	HTTPStatusCode int    `json:"httpStatusCode"`
	Response       Response
	Status         string
	Message        string
}

// HTTPBadGatewayError ...
type HTTPBadGatewayError struct {
	HTTPStatus     string `json:"httpStatus"`
	HTTPStatusCode string `json:"httpStatusCode"`
	Status         ResponseStatus
	Message        string
}

// Status returns the response Status
func (b *DataValuesResponse) Status() string { return b.b.Status }

// Description returns the description in the response
func (b *DataValuesResponse) Description() string { return b.b.Description }

// ImportCounts return the slug for the import counts
func (b *DataValuesResponse) ImportCounts() string {

	out, err := json.Marshal(b.b.ImportCount)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s", string(out))
}

// Conflicts returns the conflicts in the response
func (b *DataValuesResponse) Conflicts() string {

	out, err := json.Marshal(b.b.Conflicts)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s", string(out))
}

// IsValidDataValuesRequest return true if body is a valid DataValuesRequest
func IsValidDataValuesRequest(body string) bool {
	return true
}
