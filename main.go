package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gcinnovate/go-dispatcher2/config"
	"github.com/gcinnovate/go-dispatcher2/controllers"
	"github.com/gcinnovate/go-dispatcher2/db"
	"github.com/gcinnovate/go-dispatcher2/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// RequestObj is our object used by consumers
type RequestObj struct {
	Source             int    `db:"source"`
	Destination        int    `db:"destination"`
	Body               string `db:"body"`
	Retries            int    `db:"retries"`
	InSubmissoinPeriod bool   `db:"in_submission_period"`
	ContentType        string `db:"ctype"`
	BodyIsQueryParams  bool   `db:"body_is_query_param"`
	SubmissionID       int64  `db:"submissionid"`
	URLSurffix         string `db:"url_suffix"`
}

func produce(db *sqlx.DB, jobs chan<- int) {
	for {
		rows, err := db.Queryx(`
		SELECT 
			id 
		FROM 
			requests 
		WHERE 
			status = $1 
		ORDER BY
			created ASC LIMIT 100000
		`, "ready")
		if err != nil {
			log.Fatalln(err)
		}
		for rows.Next() {
			var requestID int
			err := rows.Scan(&requestID)
			if err != nil {
				log.Fatalln("==>", err)
			}
			fmt.Println("Adding request id", requestID)
			jobs <- requestID
			fmt.Println("Added", requestID)
		}
		fmt.Println("Fetch Requests")
		log.Printf("Going to sleep for: %v", config.Dispatcher2Conf.Server.RequestProcessInterval)
		time.Sleep(
			time.Duration(config.Dispatcher2Conf.Server.RequestProcessInterval) * time.Second)
	}
	// close(jobs)
}

func consume(db *sqlx.DB, worker int, jobs <-chan int, done chan<- bool) {
	for req := range jobs {
		// fmt.Printf("Message %v is consumed by worker %v.\n", req, worker)

		reqObj := RequestObj{}
		tx := db.MustBegin()
		err := tx.QueryRowx(`
		SELECT 
			source, 
			destination, 
			body, 
			retries, 
			in_submission_period(destination), 
			ctype,
			body_is_query_param, 
			submissionid, 
			url_suffix 
		FROM 
			requests 
		WHERE
			id = $1 
		FOR UPDATE NOWAIT`, req).StructScan(&reqObj)
		if err != nil {
			log.Fatalln(err)
		}
		tx.Commit()
		fmt.Printf("Worker:[%v] %#v\n", worker, reqObj)
	}
	done <- true
}

// Number of consumers to use when processing requests
var consumerCount int = config.Dispatcher2Conf.Server.MaxConcurrent

func main() {
	// psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s "+
	// 	"dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", config.Dispatcher2Conf.Database.URI)
	if err != nil {
		log.Fatalln(err)
	}
	/*Do proxy Stuff Here */
	go func() {
		proxyRouter := gin.Default()
		proxyRouter.Use(controllers.APIMiddleware(db))
		proxyRouter.Any("/*proxyPath", controllers.Proxy)

		proxyRouter.Run(":" + config.Dispatcher2Conf.Server.ProxyPort)
	}()

	// Now do the producer - consumer stuff
	jobs := make(chan int)
	done := make(chan bool)

	go produce(db, jobs)

	for i := 1; i <= consumerCount; i++ {
		go consume(db, i, jobs, done)
	}
	// Do the HTTP Requests Here
	router := gin.Default()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "testing",
			})
		})
	}

	v2 := router.Group("/api/v1", basicAuth())
	{
		v2.GET("/test2", func(c *gin.Context) {
			c.String(200, "Authorized")
		})
	}

	// Handle error response when a route is not defined
	router.NoRoute(func(c *gin.Context) {
		c.String(404, "Page Not Found!")
	})
	fmt.Println("Got here!")

	router.Run(":" + config.Dispatcher2Conf.Server.Port)
	<-done
}

func basicAuth() gin.HandlerFunc {

	return func(c *gin.Context) {
		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)

		if len(auth) != 2 || auth[0] != "Basic" {
			respondWithError(401, "Unauthorized", c)
			return
		}
		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !authenticateUser(pair[0], pair[1]) {
			respondWithError(401, "Unauthorized", c)
			// c.Writer.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
			return
		}

		c.Next()
	}
}

func authenticateUser(username, password string) bool {
	log.Printf("Username:%s, password:%s", username, password)
	userObj := models.User{}
	err := db.GetDB().QueryRowx(
		"SELECT id, username, name, phone, email FROM users "+
			"WHERE username = $1 AND password = crypt($2, password) ",
		username, password).StructScan(&userObj)
	if err != nil {
		fmt.Printf("User:[%v]", err)
		return false
	}
	fmt.Printf("User:[%v]", userObj)
	return true
}

func respondWithError(code int, message string, c *gin.Context) {
	resp := map[string]string{"error": message}

	c.JSON(code, resp)
	c.Abort()
}
