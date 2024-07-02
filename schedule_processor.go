package main

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"go-dispatcher2/config"
	"go-dispatcher2/models"
	"sync"
	"time"
)

const dueSchedulesSQL = `
SELECT id FROM schedules WHERE next_run_at <= NOW() AND is_active = true AND status = 'ready'
ORDER by id ASC
`

func getDueSchedules(db *sqlx.DB) ([]int64, error) {
	var ids []int64
	err := db.Select(&ids, dueSchedulesSQL)
	return ids, err
}

// ProduceSchedules a function that reads ready schedules by id from the database and sends the Id to a job channel for consumer to receive and process
func ProduceSchedules(
	db *sqlx.DB,
	jobs chan<- int64,
	wg *sync.WaitGroup,
	workingOnMutex *sync.RWMutex,
	workingOn map[int64]bool,
) {
	defer wg.Done()
	log.Info("..:::.. Starting to produce due schedules..:::..")
	dbConn, err := sqlx.Connect("postgres", config.Dispatcher2Conf.Database.URI)
	if err != nil {
		log.Fatalln("Schedule producer failed to connect to database: %v", err)
	}
	for {
		ids, err := getDueSchedules(dbConn)
		if err != nil {
			log.WithError(err).Error("Error fetching due schedules:", err)
			time.Sleep(1 * time.Minute)
			continue
		}
		schedulesCount := len(ids)
		for _, id := range ids {
			workingOnMutex.Lock()
			if !workingOn[id] {
				workingOn[id] = true
				jobs <- id
			}
			workingOnMutex.Unlock()
		}
		if schedulesCount > 0 {
			log.WithField("ScheduledCount", schedulesCount).Info("Schedules produced")
		}
		log.Info(fmt.Sprintf("Schedule producer going to sleep for: %v", config.Dispatcher2Conf.Server.RequestProcessInterval))
		time.Sleep(
			time.Duration(config.Dispatcher2Conf.Server.RequestProcessInterval) * time.Second)
	}

}

func ConsumeSchedules(db *sqlx.DB, jobs <-chan int64, wg *sync.WaitGroup, workingOnMutex *sync.RWMutex, workingOn map[int64]bool) {
	defer wg.Done()
	for id := range jobs {
		ProcessSchedule(db, id)
		workingOnMutex.Lock()
		delete(workingOn, id)
		log.WithFields(log.Fields{
			"scheduleID":    id,
			"seenMapLength": len(workingOn),
			// "senMap":        seenMap,
		}).Info("Consumer done with schedule.")
		workingOnMutex.Unlock()
		time.Sleep(1 * time.Second)
	}
}

func ProcessSchedule(db *sqlx.DB, id int64) {
	log.WithField("ScheduleID", id).Info("Processing Schedule")
	schedule, err := models.GetSchedule(db, id)
	if err != nil {
		log.WithError(err).Error("Failed to fetch schedule")
		return
	}
	tx, err := db.Beginx()
	if err != nil {
		log.Fatalln(err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	switch schedule.ScheduleType {
	case "dhis2_async_job_check":
		err = models.CheckDhis2AsyncJob(tx, schedule)
		if err != nil {
			log.WithError(err).Error("Failed to check dhis2 async job")
		}
		// Get server tied to schedule
		schedule.Status = "completed"
		schedule.Updated = time.Now().In(models.Location)
		err = models.UpdateScheduleTx(tx, schedule)
		if err != nil {
			log.WithError(err).Error("Failed to update schedule")
		}
	case "url":
		log.Info("Handling URL schedule")
	case "sms":
		log.Info("Handling URL schedule")
	case "contact_push":
		log.Info("Handling contact push schedule")
	case "command":
		log.Info("Handling command schedule")
	default:
		log.Info("Unknown schedule")

	}
}

func StartScheduleConsumers(scheduledJobs <-chan int64, wg *sync.WaitGroup, mutex *sync.RWMutex, workingOn map[int64]bool) {
	dbURI := config.Dispatcher2Conf.Database.URI
	log.Info(fmt.Sprintf("Going to create %d Schedule Consumers. Timezone: %s!!!!!\n",
		config.Dispatcher2Conf.Server.MaxConcurrent, config.Dispatcher2Conf.Server.TimeZone))
	numConsumers := 0
	for i := 1; i <= config.Dispatcher2Conf.Server.MaxConcurrent; i++ {
		newConn, err := sqlx.Connect("postgres", dbURI)
		if err != nil {
			log.WithError(err).Error("Schedule processor failed to connect to database")
		} else {
			log.Info(fmt.Sprintf("Adding Schedule Consumer: %d\n", i))
			wg.Add(1)
			go ConsumeSchedules(newConn, scheduledJobs, wg, mutex, workingOn)
			numConsumers++
		}
	}
	if numConsumers == 0 {
		log.Fatalln("Schedule processor failed to connect to database for any of the consumers")
	}
	log.Info(fmt.Sprintf("Created %d schedule jobs consumers", numConsumers))
}
