package models

import "time"

// Dhis2IndicatorMapping holds the DHIS 2 indicator mappings
type Dhis2IndicatorMapping struct {
	ID                   int64     `db:"id" json:"id"`
	UID                  string    `db:"uid" json:"uid"`
	Name                 string    `db:"name" json:"name"`
	Description          string    `db:"description" json:"description"`
	Form                 string    `db:"form" json:"form"`
	Slug                 string    `db:"slug" json:"slug"`
	Cmd                  string    `db:"cmd" json:"cmd"`
	FormOrder            string    `db:"form_order" json:"form_order"`
	DataElement          string    `db:"dataelement" json:""`
	DataSet              string    `db:"dataset" json:"dataset"`
	AttributeOptionCombo string    `db:"attribute_option_combo" json:"attribute_option_combo"`
	CategoryCombo        string    `db:"category_combo" json:"category_combo"`
	Created              time.Time `db:"created" json:"created"`
	Updated              time.Time `db:"updated" json:"updated"`
}
