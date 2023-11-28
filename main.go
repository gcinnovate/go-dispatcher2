package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go-dispatcher2/config"
	"go-dispatcher2/controllers"
	"go-dispatcher2/models"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	// log.SetFormatter(&log.JSONFormatter{})
	// log.DisableTimestamp = false
	formatter := new(log.TextFormatter)
	formatter.TimestampFormat = time.RFC3339
	formatter.FullTimestamp = true
	log.SetFormatter(formatter)
	log.SetOutput(os.Stdout)
}

var splash = `
╺┳┓╻┏━┓┏━┓┏━┓╺┳╸┏━╸╻ ╻┏━╸┏━┓┏━┓   ┏━╸┏━┓
 ┃┃┃┗━┓┣━┛┣━┫ ┃ ┃  ┣━┫┣╸ ┣┳┛┏━┛╺━╸┃╺┓┃ ┃
╺┻┛╹┗━┛╹  ╹ ╹ ╹ ┗━╸╹ ╹┗━╸╹┗╸┗━╸   ┗━┛┗━┛
`

func main() {
	fmt.Printf(splash)
	dbConn, err := sqlx.Connect("postgres", config.Dispatcher2Conf.Database.URI)
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		// Create a new scheduler
		s := gocron.NewScheduler(time.UTC)
		// Schedule the task to run "30 minutes after midn, 4am, 8am, 12pm..., everyday"

		// retrying incomplete requests runs every 5 minutes
		log.WithFields(log.Fields{"RetryCronExpression": config.Dispatcher2Conf.API.RetryCronExpression}).Info(
			"Request Retry Cron Expression")
		_, err = s.Cron(config.Dispatcher2Conf.API.RetryCronExpression).Do(RetryIncompleteRequests)
		if err != nil {
			log.WithError(err).Error("Error scheduling incomplete request retry task:")
		}
		s.StartAsync()
	}()
	/*Do proxy Stuff Here */
	go func() {
		proxyRouter := gin.Default()
		proxyRouter.Use(controllers.APIMiddleware(dbConn))
		proxyRouter.Any("/*proxyPath", controllers.Proxy)

		_ = proxyRouter.Run(":" + config.Dispatcher2Conf.Server.ProxyPort)
	}()

	jobs := make(chan int)
	var wg sync.WaitGroup

	seenMap := make(map[models.RequestID]bool)
	mutex := &sync.Mutex{}
	rWMutex := &sync.RWMutex{}

	if !*config.SkipRequestProcessing {
		// don't produce anything if skip processing is enabled

		// Start the producer goroutine
		wg.Add(1)
		go Produce(dbConn, jobs, &wg, mutex, seenMap)

		// Start the consumer goroutine
		wg.Add(1)
		go StartConsumers(jobs, &wg, rWMutex, seenMap)
	}

	// Start the backend API gin server
	wg.Add(1)
	go startAPIServer(&wg)

	wg.Wait()
}

func startAPIServer(wg *sync.WaitGroup) {
	defer wg.Done()
	router := gin.Default()
	v2 := router.Group("/api", models.BasicAuth())
	{
		v2.GET("/test2", func(c *gin.Context) {
			c.String(200, "Authorized")
		})

		q := new(controllers.QueueController)
		v2.POST("/queue", q.Queue)
		v2.GET("/queue", q.Requests)
		v2.GET("/queue/:id", q.GetRequest)
		v2.DELETE("/queue/:id", q.DeleteRequest)

		//s := new(controllers.ServerController)
		//v2.POST("/servers", s.CreateServer)
		//v2.POST("/importServers", s.ImportServers)

	}
	// Handle error response when a route is not defined
	router.NoRoute(func(c *gin.Context) {
		c.String(404, "Page Not Found!")
	})

	_ = router.Run(":" + fmt.Sprintf("%s", config.Dispatcher2Conf.Server.Port))
}
