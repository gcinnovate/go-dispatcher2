package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go-dispatcher2/utils"
)

type dhis2Payload struct {
	DataSet              string              `json:"dataSet"`
	AttributeOptionCombo string              `json:"attributeOptionCombo"`
	OrgUnit              string              `json:"orgUnit"`
	Period               string              `json:"period"`
	CompleteDate         string              `json:"completeDate,omitempty"`
	DataValues           []map[string]string `josn:"dataValues"`
}

type result struct {
	Value    string `json:"value"`
	Category string `json:"category"`
}

type postObject struct {
	Contact map[string]string `json:"contact"`
	Flow    map[string]string `json:"flow"`
	Results map[string]result `json:"results"`
}

/*
func getValue(obj map[string]interface{}, key string) string {
	for k, v := range obj {
		if k == key {
			return v.(string)
		}
	}
	return ""
}
*/

func getValue(obj map[string]string, key string) string {
	fmt.Printf("-----> %#v", obj)
	if v, ok := obj[key]; ok {
		return v
	}
	return ""
}

// RapidProController defines the rp-queue request methods
type RapidProController struct{}

// RapidProQueue method handles the /rp-queue request
func (r *RapidProController) RapidProQueue(c *gin.Context) {

	db := c.MustGet("dbConn").(*sqlx.DB)
	_, err := db.Queryx(`SELECT 1`)
	if err != nil {
		fmt.Printf("%v", err)
	}
	fmt.Printf("Got The request from DB => %v", utils.GetUID())

	var payload dhis2Payload

	var postObj postObject

	if err := c.ShouldBind(&postObj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// fmt.Printf("==========> %v", postObj)
	contact := postObj.Contact
	fmt.Printf("\n%v\n", getValue(contact, "urn"))
	payload.OrgUnit = getValue(contact, "orgunit")

	// assume the flow object passed the dataSet
	flow := postObj.Flow
	payload.DataSet = getValue(flow, "dataset")
	payload.AttributeOptionCombo = getValue(flow, "attributeOptionCombo")

	// payload.DataValues = append(payload.DataValues, map[string]string{"name": "seki"})
	// Use indicator mapping to find variables that are allowed
	fmt.Printf("%#v", payload)

	c.JSON(http.StatusOK, payload)
	return
}
