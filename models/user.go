package models

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go-dispatcher2/db"
	"strings"
	"time"
)

// User is our user object
type User struct {
	ID           int64     `db:"id" json:"id"`
	UID          string    `db:"uid" json:"uid"`
	Username     string    `db:"username" json:"username"`
	Password     string    `db:"password" json:"-"`
	FirstName    string    `db:"firstname" json:"firstname"`
	LastName     string    `db:"lastname" json:"lastname"`
	Email        string    `db:"email" json:"email"`
	Phone        string    `db:"telephone" json:"telephone"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	IsSystemUser bool      `db:"is_system_user" json:"is_system_user"`
	Created      time.Time `db:"created" json:"created"`
	Updated      time.Time `db:"updated" json:"updated"`
}

func (u *User) DeactivateAPITokens(token string) {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(
		`UPDATE user_apitoken SET is_active = FALSE WHERE user_id = :id`, u)
	if err != nil {
		log.WithError(err).Error("Failed to deactivate user API tokens")
	}
}

type UserToken struct {
	ID       int64     `db:"id" json:"id"`
	UserID   int64     `db:"user_id" json:"user_id"`
	Token    string    `db:"token" json:"token"`
	IsActive bool      `db:"is_active" json:"is_active"`
	Created  time.Time `db:"created" json:"created"`
	Updated  time.Time `db:"updated" json:"updated"`
}

func (ut *UserToken) Save() {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(`INSERT INTO user_apitoken (user_id, token)
			VALUES(:user_id, :token)`, ut)
	if err != nil {
		log.WithError(err).Error("Failed to save user API token")
	}
}

func (u *User) GetActiveToken() (string, error) {
	dbConn := db.GetDB()
	var ut UserToken
	err := dbConn.Get(&ut, "SELECT * FROM user_apitoken WHERE user_id = $1 AND is_active = TRUE LIMIT 1", u.ID)
	if err != nil {
		return "", err
	}
	return ut.Token, nil
}
func BasicAuth() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Set("dbConn", db.GetDB())
		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)

		if len(auth) != 2 || (auth[0] != "Basic" && auth[0] != "Token:") {
			RespondWithError(401, "Unauthorized", c)
			return
		}
		tokenAuthenticated, userUID := AuthenticateUserToken(auth[1])
		if auth[0] == "Token:" {
			if !tokenAuthenticated {
				RespondWithError(401, "Unauthorized", c)
				return
			}
			c.Set("currentUser", userUID)
			c.Next()
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		basicAuthenticated, userUID := AuthenticateUser(pair[0], pair[1])

		if len(pair) != 2 || !basicAuthenticated {
			RespondWithError(401, "Unauthorized", c)
			// c.Writer.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			return
		}
		c.Set("currentUser", userUID)

		c.Next()
	}
}

func GetUserByUID(uid string) (*User, error) {
	userObj := User{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, uid, username, firstname, lastname , telephone, email
        FROM users
        WHERE
            uid = $1`,
		uid).StructScan(&userObj)
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func GetUserById(id int64) (*User, error) {
	userObj := User{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, uid, username, firstname, lastname , telephone, email
        FROM users
        WHERE
            id = $1`,
		id).StructScan(&userObj)
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func AuthenticateUser(username, password string) (bool, int64) {
	// log.Printf("Username:%s, password:%s", username, password)
	userObj := User{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, uid, username, firstname, lastname , telephone, email
        FROM users
        WHERE
            username = $1 AND password = crypt($2, password)`,
		username, password).StructScan(&userObj)
	if err != nil {
		// fmt.Printf("User:[%v]", err)
		return false, 0
	}
	// fmt.Printf("User:[%v]", userObj)
	return true, userObj.ID
}

func AuthenticateUserToken(token string) (bool, int64) {
	userToken := UserToken{}
	err := db.GetDB().QueryRowx(
		`SELECT
            id, user_id, token, is_active
        FROM user_apitoken
        WHERE
            token = $1 AND is_active = TRUE LIMIT 1`,
		token).StructScan(&userToken)
	if err != nil {
		// fmt.Printf("User:[%v]", err)
		return false, 0
	}
	// fmt.Printf("User:[%v]", userObj)
	return true, userToken.UserID
}

func RespondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
	c.Abort()
}

func GenerateToken() (string, error) {
	// Define the length of the token in bytes
	const tokenLength = 20

	// Create a byte slice to hold the random bytes
	token := make([]byte, tokenLength)

	// Generate random bytes
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	// Convert the bytes to a hexadecimal string
	return hex.EncodeToString(token), nil
}
