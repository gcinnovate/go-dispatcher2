package controllers

import (
	"fmt"
	"net/http"

	"github.com/gcinnovate/go-dispatcher2/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// QueueController defines the queue request controller methods
type QueueController struct{}

// Queue method handles the /queque request
func (q *QueueController) Queue(c *gin.Context) {

	db := c.MustGet("dbConn").(*sqlx.DB)

	// source := c.PostForm("source")
	// destination := c.PostForm("destination")
	contentType := c.Request.Header.Get("Content-Type")
	models.NewRequest(c, db)

	fmt.Printf("cType %s", contentType)
	c.JSON(http.StatusOK, gin.H{"status": "Message queued"})
	return
}
