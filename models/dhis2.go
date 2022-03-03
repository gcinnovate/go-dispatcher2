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

// ImportCount is the import stats
type ImportCount struct {
	Imported string `json:"imported"`
	Updated  string `json:"updated"`
	Deleted  string `json:"deleted"`
	Ignored  string `json:"ignored"`
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
