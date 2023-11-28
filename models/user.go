package models

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"go-dispatcher2/db"
	"strings"
	"time"
)

// User is our user object
type User struct {
	ID           int64     `db:"id"`
	UID          string    `db:"uid" json:"uid"`
	Username     string    `db:"username"`
	Password     string    `db:"password"`
	FirstName    string    `json:"firstname" db:"firstname"`
	LastName     string    `json:"lastname" db:"lastname"`
	Email        string    `json:"email" db:"email"`
	Phone        string    `json:"telephone" db:"telephone"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	IsSystemUser bool      `json:"is_system_user" db:"is_system_user"`
	Created      time.Time `json:"created" db:"created"`
	Updated      time.Time `json:"updated" db:"updated"`
}

func BasicAuth() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Set("dbConn", db.GetDB())
		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)

		if len(auth) != 2 || auth[0] != "Basic" {
			RespondWithError(401, "Unauthorized", c)
			return
		}
		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !AuthenticateUser(pair[0], pair[1]) {
			RespondWithError(401, "Unauthorized", c)
			// c.Writer.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			return
		}

		c.Next()
	}
}

func AuthenticateUser(username, password string) bool {
	// log.Printf("Username:%s, password:%s", username, password)
	userObj := User{}
	err := db.GetDB().QueryRowx(
		`SELECT
                        id, username, firstname, lastname , telephone, email
                FROM users
                WHERE
                        username = $1 AND password = crypt($2, password)`,
		username, password).StructScan(&userObj)
	if err != nil {
		// fmt.Printf("User:[%v]", err)
		return false
	}
	// fmt.Printf("User:[%v]", userObj)
	return true
}

func RespondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
	c.Abort()
}
