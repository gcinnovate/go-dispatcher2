package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"go-dispatcher2/config"
	"go-dispatcher2/db"
	"go-dispatcher2/models"
	"go-dispatcher2/utils/dbutils"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ServerStatus is the status object within each request to
// track status for the CC servers
type ServerStatus struct {
	Retries    int                  `json:"retries"`
	Status     models.RequestStatus `json:"status,omitempty"`
	StatusCode string               `json:"statuscode,omitempty"`
	Response   string               `json:"response,omitempty"`
	Errors     string               `json:"errors"`
}

// AddParamsToURL takes a URL and add extra parameters to it from dbutils.MapAnything
// check whether URL doesn't contain ? at the end before adding parameters, if so simply add parameters
func AddParamsToURL(myURL string, params dbutils.MapAnything) string {
	if !strings.HasSuffix(myURL, "?") {
		myURL = myURL + "?"
	}
	p := url.Values{}
	for k, v := range params {
		p.Add(k, fmt.Sprintf("%v", v))
	}
	return myURL + p.Encode()
}

// Scan is the db driver scanner for ServerStatus
func (a *ServerStatus) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// RequestObject is our object used by consumers
type RequestObject struct {
	ID                 models.RequestID     `db:"id"`
	Source             int                  `db:"source"`
	Destination        int                  `db:"destination"`
	DependsOn          dbutils.Int          `db:"depends_on"`
	CCServers          pq.Int32Array        `db:"cc_servers" json:"CCServers"`
	CCServersStatus    dbutils.MapAnything  `db:"cc_servers_status" json:"CCServersStatus"`
	Body               string               `db:"body"`
	Response           string               `db:"response"`
	Retries            int                  `db:"retries"`
	InSubmissionPeriod bool                 `db:"in_submission_period"`
	ContentType        string               `db:"content_type"`
	ObjectType         string               `db:"object_type"`
	BodyIsQueryParams  bool                 `db:"body_is_query_param"`
	SubmissionID       string               `db:"submissionid"`
	URLSurffix         string               `db:"url_suffix"`
	Suspended          bool                 `db:"suspended"`
	Status             models.RequestStatus `db:"status"`
	StatusCode         string               `db:"statuscode"`
	Errors             string               `db:"errors"`
}

const updateRequestSQL = `
UPDATE requests SET (status, statuscode, errors, retries, response, updated)
	= (:status, :statuscode, :errors, :retries, :response, current_timestamp) WHERE id = :id
`
const updateStatusSQL = `
	UPDATE requests SET (status,  updated) = (:status, current_timestamp)
	WHERE id = :id`

const selectRequestObjectSQL = `
SELECT id, source, destination, depends_on, cc_servers, cc_servers_status, body, 
	response, retries, ctype, object_type, body_is_query_param, submissionid, 
	url_suffix, suspended, status, statuscode, errors
FROM requests WHERE id = $1;`

// HasDependency returns true if request has a request it depends on
func (r *RequestObject) HasDependency() bool {
	return r.DependsOn > 0
}

// DependencyCompleted returns true request's dependent request was completed
func (r *RequestObject) DependencyCompleted(tx *sqlx.Tx) bool {
	if r.HasDependency() {
		completed := false
		err := tx.Get(&completed, "SELECT status = 'completed' FROM requests WHERE id = $1", r.DependsOn)
		if err != nil {
			log.WithError(err).Info("Error reading dependent request status")
			return false
		}
		return completed
	}
	return false
}

// GetRequestObjectById returns the requested object
func GetRequestObjectById(db *sqlx.DB, id models.RequestID) (*RequestObject, error) {
	var request RequestObject
	err := db.Get(&request, selectRequestObjectSQL, id)
	if err != nil {
		return &RequestObject{}, err
	}
	return &request, nil
}

// updateRequest is used by consumers to update request in the db
func (r *RequestObject) updateRequest(tx *sqlx.Tx) {
	_, err := tx.NamedExec(updateRequestSQL, r)
	if err != nil {
		log.WithError(err).Error("Error updating request status")
	}

	// _ = db.Commit()
}

// updateCCServerStatus updates the status for CC servers on the request
func (r *RequestObject) updateCCServerStatus(tx *sqlx.Tx) {
	_, err := tx.NamedExec(`UPDATE requests SET cc_servers_status = :cc_servers_status WHERE id = :id`, r)
	if err != nil {
		log.WithError(err).Error("Error updating request CC Server Status!")
	}
	log.WithFields(log.Fields{"ReqID": r.ID, "ServerStatus": r.CCServersStatus}).Info(">>>>>>>>>>>>>>")
}

// updateRequestStatus
func (r *RequestObject) updateRequestStatus(tx *sqlx.Tx) {
	_, err := tx.NamedExec(updateStatusSQL, r)
	if err != nil {
		log.WithError(err).Error("Error updating request")
	}
}

// WithStatus updates the RequestObj status with passed value
func (r *RequestObject) WithStatus(s models.RequestStatus) *RequestObject { r.Status = s; return r }

// canSendRequest checks if a queued request is eligible for sending
// based on constraints on request and the receiving servers
func (r *RequestObject) canSendRequest(tx *sqlx.Tx, server models.Server, serverInCC bool) bool {
	reason := ""
	log.WithField("Reason", reason)

	if r.HasDependency() {
		if !r.DependencyCompleted(tx) {
			reason = "Dependency incomplete."
			return false
		}
	}
	if !serverInCC {
		// check if we have exceeded retries
		if r.Retries > config.Dispatcher2Conf.Server.MaxRetries {
			reason = "Max retries exceeded."
			r.Status = models.RequestStatusExpired
			r.updateRequestStatus(tx)
			log.WithFields(log.Fields{
				"requestID": r.ID,
				"retries":   r.Retries,
				"reason":    reason,
			}).Info("Cannot send request")
			return false
		}
		// check if we're  suspended
		if server.Suspended() {
			reason = "Destination server is suspended."
			log.WithFields(log.Fields{
				"server": server.ID(),
				"name":   server.Name(),
			}).Info("Destination server is suspended")
			return false
		}
		// check if we're out of submission period
		if !r.InSubmissionPeriod {
			reason = "Destination server out of submission period."
			log.WithFields(log.Fields{
				"server": server.ID,
				"name":   server.Name,
			}).Info("Destination server out of submission period")
			return false
		}
		// check if this request is  blacklisted
		if r.Suspended {
			reason = "Request blacklisted."
			r.Errors = "Blacklisted"
			r.StatusCode = "ERROR7"
			r.Retries += 1
			r.Status = models.RequestStatusCanceled
			r.updateRequest(tx)
			log.WithFields(log.Fields{
				"request": r.ID,
			}).Info("Request blacklisted")
			return false
		}
		// check if body is empty
		if len(strings.TrimSpace(r.Body)) == 0 {
			reason = "Request has empty body."
			r.Status = models.RequestStatusFailed
			r.StatusCode = "ERROR1"
			r.Errors = "Request has empty body"
			r.updateRequest(tx)
			log.WithFields(log.Fields{
				"request": r.ID,
			}).Info("Request has empty body")
			return false
		}
		return true
	} else {
		// if ccServerStatus := r.CCServersStatus[server];
		// lo.Filter()
		ccServers := lo.Filter(r.CCServers, func(item int32, index int) bool {
			if item == int32(server.ID()) && item != int32(r.Destination) {
				// just make sure we don't sent to cc server same as destination on request
				return true
			}
			return false
		})
		if len(ccServers) > 0 {
			var ccServerStatus ServerStatus
			if ccServerObject, ok := models.ServerMap[fmt.Sprintf("%d", ccServers[0])]; ok {
				// Check if cc server is suspended
				if ccServerObject.Suspended() {
					return false
				}
				// get server status from request
				if ccstatusObj, ok := r.CCServersStatus[fmt.Sprintf("%d", ccServerObject.ID())]; ok {

					if val, ok := ccstatusObj.(ServerStatus); ok {
						ccServerStatus = val
					}
				}
				// Now check with the ccServerStatus object for sending eligibility
				// check if we have exceeded the retries for this server
				if ccServerStatus.Retries > config.Dispatcher2Conf.Server.MaxRetries {
					ccServerStatus.Status = models.RequestStatusExpired
					var ccServerStatusJSON dbutils.MapAnything
					err := ccServerStatusJSON.Scan(ccServerStatus)
					if err != nil {
						log.WithError(err).Error("Failed to convert CC server status to required db type")
						return false
					}
					r.CCServersStatus = ccServerStatusJSON
					r.updateCCServerStatus(tx)
					return false
				}
				// check if we're out of submission period
				if !ccServerObject.InSubmissionPeriod(tx) {
					log.WithFields(log.Fields{
						"server": ccServerObject.ID,
						"name":   ccServerObject.Name,
					}).Info("Destination server out of submission period")
					return false
				}

				// check if we're  suspended
				if ccServerObject.Suspended() {
					ccServerStatus.Errors = "Blacklisted"
					ccServerStatus.StatusCode = "ERROR7"
					ccServerStatus.Retries += 1
					ccServerStatus.Status = models.RequestStatusCanceled
					var ccServerStatusJSON dbutils.MapAnything
					err := ccServerStatusJSON.Scan(ccServerStatus)
					if err != nil {
						log.WithError(err).Error("Failed to convert CC server status to required db type")
						return false
					}
					r.CCServersStatus = ccServerStatusJSON
					r.updateCCServerStatus(tx)
					log.WithFields(log.Fields{
						"server": ccServerObject.ID,
						"name":   ccServerObject.Name,
					}).Info("Destination server is suspended")
					return false
				}
				// check if this request is  blacklisted
				if r.Suspended {
					ccServerStatus.Errors = "Blacklisted"
					ccServerStatus.StatusCode = "ERROR7"
					ccServerStatus.Retries += 1
					ccServerStatus.Status = models.RequestStatusCanceled
					var ccServerStatusJSON dbutils.MapAnything
					err := ccServerStatusJSON.Scan(ccServerStatus)
					if err != nil {
						log.WithError(err).Error("Failed to convert CC server status to required db type")
						return false
					}
					r.CCServersStatus = ccServerStatusJSON
					r.updateCCServerStatus(tx)
					log.WithFields(log.Fields{
						"request": r.ID, "CCServer": ccServers[0],
					}).Info("Request blacklisted for CC Server")
					return false
				}

				// check if body is empty
				if len(strings.TrimSpace(r.Body)) == 0 {
					ccServerStatus.Status = models.RequestStatusFailed
					ccServerStatus.StatusCode = "ERROR1"
					ccServerStatus.Errors = "Request has empty body"
					var ccServerStatusJSON dbutils.MapAnything
					err := ccServerStatusJSON.Scan(ccServerStatus)
					if err != nil {
						log.WithError(err).Error("Failed to convert CC server status to required db type")
						return false
					}
					r.CCServersStatus = ccServerStatusJSON
					r.updateCCServerStatus(tx)

					log.WithFields(log.Fields{
						"request": r.ID, "CCServer": ccServerObject.ID,
					}).Info("Request has empty body")
					return false
				}
				return true

			}
		}

		return true
	}

}

func (r *RequestObject) unMarshalBody() (interface{}, error) {
	var data interface{}
	switch r.ObjectType {
	case "ORGANISATION_UNITS":
		//data = models.DataValuesRequest{}
		//err := json.Unmarshal([]byte(r.Body), &data)
		//if err != nil {
		//	return nil, err
		//}
	default:
		data = map[string]interface{}{}
		err := json.Unmarshal([]byte(r.Body), &data)
		if err != nil {
			return nil, err
		}

	}
	return data, nil
}

// sendRequest sends request to destination server
func (r *RequestObject) sendRequest(destination models.Server) (*http.Response, error) {
	data, err := r.unMarshalBody()
	if err != nil {
		return nil, err
	}
	marshalled, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Failed to marshal request body")
		return nil, err
	}
	destURL := destination.URL()
	if len(r.URLSurffix) > 1 {
		destURL += r.URLSurffix
	}
	completeURL := AddParamsToURL(destURL, destination.URLParams())
	log.WithFields(log.Fields{
		"request": r.ID,
		"server":  destination.ID(),
		"url":     completeURL,
	}).Info("Sending request to destination server")
	req, err := http.NewRequest(destination.HTTPMethod(), completeURL, bytes.NewReader(marshalled))

	switch destination.AuthMethod() {
	case "Token":
		// Add API token
		tokenAuth := "ApiToken " + destination.AuthToken()
		req.Header.Set("Authorization", tokenAuth)
		log.WithField("AuthToken", tokenAuth).Info("The authentication token:")
	default: // Basic Auth
		// Add basic authentication
		auth := destination.Username() + ":" + destination.Password()
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", basicAuth)

	}

	req.Header.Set("Content-Type", r.ContentType)
	// Create custom transport with TLS settings
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			// Set any necessary TLS settings here
			// For example, to disable certificate validation:
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// var RequestsMap = make(map[string]int)

// Produce gets all the ready requests in the queue
func Produce(db *sqlx.DB, jobs chan<- int, wg *sync.WaitGroup, mutex *sync.Mutex, seenMap map[models.RequestID]bool) {
	defer wg.Done()
	log.Println("Producer staring:!!!")

	// RequestsMap[""] = 6
	for {

		log.Println("Going to read requests")
		rows, err := db.Queryx(`
                SELECT 
                    id FROM requests 
                WHERE status = $1  and status_of_dependence(id) IN ('completed', '') 
                ORDER BY depends_on desc, created LIMIT 100000
                `, "ready")
		if err != nil {
			log.WithError(err).Error("ERROR READING READY REQUESTS!!!")
		}
		requestsCount := 0
		for rows.Next() {

			requestsCount += 1
			var requestID int
			err := rows.Scan(&requestID)
			if err != nil {
				// log.Fatalln("==>", err)
				log.WithError(err).Error("Error reading request from queue:")
			}
			mutex.Lock()
			if _, exists := seenMap[models.RequestID(requestID)]; exists {
				log.WithField("requestID", requestID).Info("Request already in dynamic queue")
				continue
			}
			mutex.Unlock()
			go func(req int) {
				// Let see if we can recover from panics XXX
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("Recovered in Produce", r)

					}
				}()
				mutex.Lock()
				defer mutex.Unlock()
				//if _, exists := seenMap[models.RequestID(req)]; exists {
				//	log.WithField("requestID", req).Info("Request already in dynamic queue")
				//	return
				//}
				jobs <- req
				seenMap[models.RequestID(req)] = true
				log.Info(fmt.Sprintf("Added Request [id: %v]", req))
			}(requestID)

		}
		if err := rows.Err(); err != nil {
			log.WithError(err).Error("Error reading requests")
		}
		_ = rows.Close()

		if requestsCount > 0 {
			log.WithField("requestsAdded", requestsCount).Info("Fetched Requests")
		}
		log.Info(fmt.Sprintf("Requests producer going to sleep for: %v", config.Dispatcher2Conf.Server.RequestProcessInterval))
		// Not good enough but let's bare with the sleep this initial version
		time.Sleep(
			time.Duration(config.Dispatcher2Conf.Server.RequestProcessInterval) * time.Second)
	}
}

// Consume is the consumer go routine
func Consume(db *sqlx.DB, worker int, jobs <-chan int, wg *sync.WaitGroup, mutex *sync.RWMutex, seenMap map[models.RequestID]bool) {
	defer wg.Done()
	fmt.Println("Calling Consumer")

	for req := range jobs {
		fmt.Printf("Message %v is consumed by worker %v.\n", req, worker)

		reqObj := RequestObject{}
		tx := db.MustBegin()
		err := tx.QueryRowx(`
                SELECT
                        id, depends_on,source, destination, cc_servers, cc_servers_status, body, retries, in_submission_period(destination),
                        content_type, object_type, body_is_query_param, submissionid, url_suffix,suspended,
                        statuscode, status, errors
                        
                FROM requests
                WHERE id = $1 FOR UPDATE NOWAIT`, req).StructScan(&reqObj)
		if err != nil {
			log.WithError(err).Error("Error reading request for processing")
			return
		}
		log.WithFields(log.Fields{
			"worker":    worker,
			"requestID": req}).Info("Handling Request")
		/* Work on the request */
		// dest = utils.GetServer(reqObj.Destination)
		// log.WithFields(log.Fields{"servers": models.ServerMap}).Info("Servers")
		if reqDestination, ok := models.ServerMap[fmt.Sprintf("%d", reqObj.Destination)]; ok {
			_ = ProcessRequest(tx, reqObj, reqDestination, false, false)

			lo.Map(reqObj.CCServers, func(item int32, index int) error {
				if ccServer, ok := models.ServerMap[fmt.Sprintf("%d", item)]; ok {
					log.WithFields(log.Fields{"CCServerID": item, "ServerIndex=>": index}).Info("!CC Server:")
					return ProcessRequest(tx, reqObj, ccServer, true, false)
				} else {
					log.WithField("ServerID", item).Info("Sever not in Map")
				}
				return nil
			})
		} else {
			// Using Go lodash to process
			lo.Map(reqObj.CCServers, func(item int32, index int) error {
				log.WithFields(log.Fields{"CCServerID": item, "ServerIndex==>": index}).Info("!!CC Server:")
				if ccServer, ok := models.ServerMap[fmt.Sprintf("%d", item)]; ok {
					return ProcessRequest(tx, reqObj, ccServer, true, false)
				} else {
					log.WithField("ServerID", item).Info("Sever not in Map>")

				}
				return nil
			})
		}

		err = tx.Commit()
		if err != nil {
			log.WithError(err).Error("Failed to Commit transaction after processing!")
		}
		mutex.Lock()
		delete(seenMap, models.RequestID(req))
		log.WithFields(log.Fields{
			"requestID":     req,
			"seenMapLength": len(seenMap),
			// "senMap":        seenMap,
		}).Info("Consumer done with request.")
		mutex.Unlock()
		// delete(RequestsMap, fmt.Sprintf("Req-%d", req))
		time.Sleep(1 * time.Second)
	}

}

// ProcessRequest handles a ready request
func ProcessRequest(tx *sqlx.Tx, reqObj RequestObject, destination models.Server, serverInCC, skipCheck bool) error {
	if skipCheck || reqObj.canSendRequest(tx, destination, serverInCC) {
		log.WithFields(log.Fields{"requestID": reqObj.ID}).Info("Request can be processed")
		// send request
		resp, err := reqObj.sendRequest(destination)
		if err != nil {
			log.WithError(err).WithField("RequestID", reqObj.ID).Error(
				"Failed to send request")
			reqObj.Status = models.RequestStatusFailed
			reqObj.StatusCode = "ERROR02"
			reqObj.Errors = "Server possibly unreachable"
			reqObj.Retries += 1
			reqObj.updateRequest(tx)
			return err
		}

		if !destination.UseAsync() {
			result := models.ImportSummary{}
			respBody, _ := io.ReadAll(resp.Body)
			err := json.Unmarshal(respBody, &result)
			// err := json.NewDecoder(resp.Body).Decode(&result)
			if err != nil {
				if serverInCC {
					serverStatus := reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())].(map[string]interface{})
					summary := "Failed to decode import summary"
					newServerStatus := make(map[string]interface{})
					newServerStatus["errors"] = summary
					newServerStatus["status"] = models.RequestStatusFailed
					newServerStatus["statusCode"] = "ERROR03"
					newServerStatus["retries"] = int(serverStatus["retries"].(float64) + 1)
					reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())] = newServerStatus
					reqObj.updateCCServerStatus(tx)
					// _, _ = tx.NamedExec(`UPDATE requests SET cc_servers_status = :cc_servers_status WHERE id = :id`, reqObj)
				} else {
					reqObj.Status = models.RequestStatusFailed
					reqObj.StatusCode = "ERROR03"
					reqObj.Errors = "Failed to decode import summary"
					reqObj.Retries += 1
					reqObj.updateRequest(tx)
					log.WithField("Resp", string(respBody)).WithError(err).Error("Failed to decode import summary")
					return err
				}
			}
			if resp.StatusCode/100 == 2 {

				if serverInCC {
					serverStatus := reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())].(map[string]interface{})
					summary := fmt.Sprintf("Created: %d, Updated: %d", result.Response.Stats.Created, result.Response.Stats.Updated)
					newServerStatus := make(map[string]interface{})
					newServerStatus["errors"] = summary
					newServerStatus["status"] = models.RequestStatusCompleted
					newServerStatus["statusCode"] = fmt.Sprintf("%d", resp.StatusCode)
					newServerStatus["retries"] = serverStatus["retries"]
					reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())] = newServerStatus
					// reqObj.updateCCServerStatus(tx)
					_, _ = tx.NamedExec(`UPDATE requests SET cc_servers_status = :cc_servers_status WHERE id = :id`, reqObj)

				} else {
					summary := fmt.Sprintf("Created: %d, Updated: %d", result.Response.Stats.Created, result.Response.Stats.Updated)
					reqObj.StatusCode = fmt.Sprintf("%d", resp.StatusCode)
					reqObj.Errors = summary
					reqObj.Retries += 1
					reqObj.Status = models.RequestStatusCompleted
					reqObj.updateRequest(tx)
					reqObj.WithStatus(models.RequestStatusCompleted).updateRequestStatus(tx)
				}
				log.WithFields(log.Fields{
					"status":     result.Response.Status,
					"created":    result.Response.Stats.Created,
					"updated":    result.Response.Stats.Updated,
					"total":      result.Response.Stats.Total,
					"serverDBId": destination.ID(),
					"requestID":  reqObj.ID,
					// "response": string(respBody),
				}).Info("Request completed successfully!")
				// reqObj.CCServersStatus.Scan()
				return nil
			} else {
				log.WithFields(log.Fields{
					"requestID": reqObj.ID, "responseStatus": resp.StatusCode, "ServerInCC": serverInCC,
				}).Warn("A non 200 response")
				if serverInCC {
					serverStatus := reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())].(map[string]interface{})
					// summary := fmt.Sprintf("Created: 0, Updated: 0")
					newServerStatus := make(map[string]interface{})
					newServerStatus["status"] = "failed"
					newServerStatus["statusCode"] = fmt.Sprintf("%d", resp.StatusCode)
					switch serverStatus["retries"].(type) {
					case float64:
						newServerStatus["retries"] = int(serverStatus["retries"].(float64) + 1)
					case int:
						newServerStatus["retries"] = serverStatus["retries"].(int) + 1
					}
					newServerStatus["errors"] = "server possibly unreachable"
					reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())] = newServerStatus
					_, _ = tx.NamedExec(`UPDATE requests SET cc_servers_status = :cc_servers_status WHERE id = :id`, reqObj)

					// reqObj.updateCCServerStatus(tx)
				} else {
					reqObj.StatusCode = fmt.Sprintf("%d", resp.StatusCode)
					reqObj.Status = models.RequestStatusFailed
					reqObj.Errors = "request might have conflicts"
					reqObj.Retries += 1
					reqObj.Response = string(respBody)
					reqObj.updateRequest(tx)
					// reqObj.withStatus(models.RequestStatusFailed).updateRequestStatus(tx)
				}
			}
		} else {
			// We are using Async

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				reqObj.WithStatus(models.RequestStatusFailed).updateRequestStatus(tx)
				log.WithError(err).Error("Could not read response")
				return err
			}
			log.WithField("responseBytes", string(bodyBytes)).Info("Response Payload")
			if resp.StatusCode/100 == 2 {
				// v, _, _, err := jsonparser.Get(bodyBytes, "status")
				v := gjson.Get(string(bodyBytes), "status").String()
				if err != nil {
					log.WithError(err).Error("No status field found by jsonparser")
				}
				fmt.Println(v)
				// jobId, err := jsonparser.GetString(bodyBytes, "response", "id")
				jobId := gjson.Get(string(bodyBytes), "response.id").String()
				if err != nil {
					log.WithError(err).Error("No job id found by jsonparser in asyn response")
					return err
				}
				// jobType, _ := jsonparser.GetString(bodyBytes, "response", "jobType")
				jobType := gjson.Get(string(bodyBytes), "response.jobType").String()
				// Create Async Schedule
				scheduleId, err := models.CreateAsyncJobSchedule(
					tx, reqObj.ID, destination.ID(), serverInCC, jobType, jobId)
				if err != nil {
					log.WithError(err).Error("Failed to create async job schedule")
					return err
				}
				log.WithFields(log.Fields{
					"scheduleID": scheduleId, "requestID": reqObj.ID}).Info("Created Async Job Schedule")
				if serverInCC {
					serverStatus := reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())].(map[string]interface{})
					summary := fmt.Sprintf("Async job sent to server")
					newServerStatus := make(map[string]interface{})
					newServerStatus["errors"] = summary
					newServerStatus["status"] = models.RequestStatusCompleted
					newServerStatus["statusCode"] = fmt.Sprintf("%d", resp.StatusCode)
					switch serverStatus["retries"].(type) {
					case float64:
						newServerStatus["retries"] = int(serverStatus["retries"].(float64) + 1)
					case int:
						newServerStatus["retries"] = serverStatus["retries"].(int) + 1
					}
					reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())] = newServerStatus
					// reqObj.updateCCServerStatus(tx)
					_, _ = tx.NamedExec(`UPDATE requests SET cc_servers_status = :cc_servers_status WHERE id = :id`, reqObj)

				} else {
					summary := fmt.Sprintf("Async job sent to server")
					reqObj.StatusCode = fmt.Sprintf("%d", resp.StatusCode)
					reqObj.Errors = summary
					reqObj.Retries += 1
					reqObj.Status = models.RequestStatusCompleted
					reqObj.updateRequest(tx)
					reqObj.WithStatus(models.RequestStatusCompleted).updateRequestStatus(tx)
				}
			} else {
				log.WithFields(log.Fields{
					"requestID": reqObj.ID, "responseStatus": resp.StatusCode, "ServerInCC": serverInCC,
				}).Warn("A non 200 response from async request")

				if serverInCC {
					serverStatus := reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())].(map[string]interface{})
					newServerStatus := make(map[string]interface{})
					newServerStatus["status"] = "failed"
					newServerStatus["statusCode"] = fmt.Sprintf("%d", resp.StatusCode)
					switch serverStatus["retries"].(type) {
					case float64:
						newServerStatus["retries"] = int(serverStatus["retries"].(float64) + 1)
					case int:
						newServerStatus["retries"] = serverStatus["retries"].(int) + 1
					}
					newServerStatus["errors"] = "server possibly unreachable"
					reqObj.CCServersStatus[fmt.Sprintf("%d", destination.ID())] = newServerStatus
					_, _ = tx.NamedExec(`UPDATE requests SET cc_servers_status = :cc_servers_status WHERE id = :id`, reqObj)

				} else {
					reqObj.StatusCode = fmt.Sprintf("%d", resp.StatusCode)
					reqObj.Status = models.RequestStatusFailed
					reqObj.Errors = "request might have conflicts while async request"
					reqObj.Retries += 1
					reqObj.Response = string(bodyBytes)
					reqObj.updateRequest(tx)
				}

			}

		}
		err = resp.Body.Close()
		if err != nil {
			log.WithError(err).Error("Failed to close response body")
		}
	} else {
		log.WithFields(log.Fields{
			"requestID": reqObj.ID,
			"skipCheck": skipCheck,
		}).Info("Cannot process request now!")
	}
	return nil
}

// StartConsumers starts the consumer go routines
func StartConsumers(jobs <-chan int, wg *sync.WaitGroup, mutex *sync.RWMutex, seedMap map[models.RequestID]bool) {
	defer wg.Done()

	dbURI := config.Dispatcher2Conf.Database.URI

	log.Info(fmt.Sprintf("Going to create %d Consumers!!!!!\n", config.Dispatcher2Conf.Server.MaxConcurrent))
	for i := 1; i <= config.Dispatcher2Conf.Server.MaxConcurrent; i++ {

		newConn, err := sqlx.Connect("postgres", dbURI)
		if err != nil {
			log.Fatalln("Request processor failed to connect to database: %v", err)
		}
		log.Info(fmt.Sprintf("Adding Request Consumer: %d\n", i))
		wg.Add(1)
		go Consume(newConn, i, jobs, wg, mutex, seedMap)
	}
	log.WithFields(log.Fields{"MaxConsumers": config.Dispatcher2Conf.Server.MaxConcurrent}).Info("Created Consumers: ")
}

const incompleteRequestsSQL = `
	SELECT id, destination, status, retries, failed_cc_servers(cc_servers, cc_servers_status) AS cc_servers, body,
	       url_suffix, cc_servers_status, object_type, content_type, body_is_query_param
	FROM requests 
	WHERE 
	    ((status IN ('completed', 'failed') AND failed_cc_servers(cc_servers, cc_servers_status) <> '{}')  
	   	OR status = 'failed') AND suspended = 0 AND status <> 'expired' ORDER by depends_on desc;
`

// RetryIncompleteRequests is intended to occasionally retry incomplete requests - there could be a success chance
// this could be scheduled to run every so often
func RetryIncompleteRequests() {
	log.Info("..::::::.. Starting to process Incomplete Requests ..::::::..")
	dbConn := db.GetDB()
	rows, err := dbConn.Queryx(incompleteRequestsSQL)
	if err != nil {
		log.WithError(err).Error("ERROR READING PREVIOUSLY INCOMPLETE REQUESTS!!!")
		return
	}

	for rows.Next() {
		reqObj := RequestObject{}
		err := rows.StructScan(&reqObj)
		if err != nil {
			log.WithError(err).Error("Error reading incomplete request for processing")
		}
		log.WithFields(log.Fields{
			"requestID": reqObj.ID}).Info("Handling Incomplete Request")
		tx := dbConn.MustBegin()

		if reqObj.Status == "failed" { // destination server request had failed
			if reqDestination, ok := models.ServerMap[fmt.Sprintf("%d", reqObj.Destination)]; ok {
				if reqObj.Retries <= config.Dispatcher2Conf.Server.MaxRetries {
					_ = ProcessRequest(tx, reqObj, reqDestination, false, true)
				} else {
					reqObj.WithStatus(models.RequestStatusExpired).updateRequestStatus(tx)
				}

				lo.Map(reqObj.CCServers, func(item int32, index int) error {
					if ccServer, ok := models.ServerMap[fmt.Sprintf("%d", item)]; ok {
						log.WithFields(log.Fields{"CCServerID": item, "ServerIndex": index}).Info(
							"- Incomplete Request Retry:")
						return ProcessRequest(tx, reqObj, ccServer, true, true)
					} else {
						log.WithField("ServerID", item).Info("Incomplete Request Retry: Sever not in Map")
					}
					return nil
				})
			}
		} else {
			lo.Map(reqObj.CCServers, func(item int32, index int) error {
				if ccServer, ok := models.ServerMap[fmt.Sprintf("%d", item)]; ok {
					log.WithFields(log.Fields{"CCServerID": item, "ServerIndex": index}).Info(
						"+ Incomplete Request Retry")
					// get cc server's status
					var ccServerStatus ServerStatus
					if ccstatusObj, ok := reqObj.CCServersStatus[fmt.Sprintf("%d", item)]; ok {

						if val, ok := ccstatusObj.(ServerStatus); ok {
							ccServerStatus = val
						}
					}

					// only retry if max retries is not exceeded else expre request
					if ccServerStatus.Retries <= config.Dispatcher2Conf.Server.MaxRetries {
						return ProcessRequest(tx, reqObj, ccServer, true, true)
					} else {
						reqObj.WithStatus(models.RequestStatusExpired).updateRequestStatus(tx)
						return nil
					}
				} else {
					log.WithField("ServerID", item).Info("Incomplete Request Retry: Sever not in Map")
				}
				return nil
			})
		}

		err = tx.Commit()
		if err != nil {
			log.WithError(err).Error("Failed to Commit transaction after processing incomplete request!")
		}
	}
	_ = rows.Close()

	log.Info("..:::.. Finished to process incomplete requests ..:::..")
}
