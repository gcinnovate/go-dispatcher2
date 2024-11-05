package main

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
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
	LoadServersFromConfigFiles(config.ServersConfigMap)

	go func() {
		// retrying incomplete requests runs every 5 minutes
		log.WithFields(log.Fields{"RetryCronExpression": config.Dispatcher2Conf.Server.RetryCronExpression}).Info(
			"Request Retry Cron Expression")
		// Create a new scheduler
		c := cron.New()
		_, err := c.AddFunc(config.Dispatcher2Conf.Server.RetryCronExpression, func() {
			RetryIncompleteRequests()
		})
		if err != nil {
			log.WithError(err).Error("Error scheduling incomplete request retry task:")
		}
		c.Start()

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
	scheduledJobs := make(chan int64)
	workingOn := make(map[int64]bool)
	var workingOnMutex = &sync.Mutex{}
	var rWworkingOnMutex = &sync.RWMutex{}

	if !*config.SkipScheduleProcessing {
		wg.Add(1)
		go ProduceSchedules(dbConn, scheduledJobs, &wg, workingOnMutex, workingOn)

		wg.Add(1)
		go StartScheduleConsumers(scheduledJobs, &wg, rWworkingOnMutex, workingOn)

	}

	// Start the backend API gin server
	wg.Add(1)
	go startAPIServer(&wg)

	wg.Wait()
	close(scheduledJobs)
	close(jobs)
}

func startAPIServer(wg *sync.WaitGroup) {
	defer wg.Done()
	router := gin.Default()
	v2 := router.Group("/api", models.BasicAuth())
	{
		v2.GET("/test2", func(c *gin.Context) {
			c.String(200, "Authorized")
		})

		tk := new(controllers.TokenController)
		v2.GET("/getToken", tk.GetActiveToken)
		v2.GET("/generateToken", tk.GenerateNewToken)
		v2.DELETE("/deleteTokens", tk.DeleteInactiveTokens)

		q := new(controllers.QueueController)
		v2.POST("/queue", q.Queue)
		v2.GET("/queue", q.Requests)
		v2.GET("/queue/:id", q.GetRequest)
		v2.DELETE("/queue/:id", q.DeleteRequest)

		//s := new(controllers.ServerController)
		//v2.POST("/servers", s.CreateServer)
		//v2.POST("/importServers", s.ImportServers)
		s := new(controllers.ScheduleController)
		v2.GET("/schedules", s.ListSchedules)
		v2.POST("/schedules", s.NewSchedule)
		v2.GET("/schedules/:id", s.GetSchedule)
		v2.POST("/schedules/:id", s.UpdateSchedule)
		v2.DELETE("/schedules/:id", s.DeleteSchedule)

	}
	// Handle error response when a route is not defined
	router.NoRoute(func(c *gin.Context) {
		c.String(404, "Page Not Found!")
	})

	_ = router.Run(":" + fmt.Sprintf("%s", config.Dispatcher2Conf.Server.Port))
}
