package handlers

import (
	"Assignment2/consts"
	"Assignment2/fsutils"
	"Assignment2/util"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// StartTime for calculating service uptime
var StartTime = time.Now()

// ServiceStatus for storage of status data before encoding to json
type ServiceStatus struct {
	CountriesApi    string `json:"countries_api"`
	NotificationsDb string `json:"notification_db"`
	Webhooks        string `json:"webhooks"`
	Version         string `json:"version"`
	Uptime          int    `json:"uptime"`
}

// Collection with one document, to check if db is available:
const dbCheckCollection = "dbCheckCollection"
const dbCheckDocument = "dbCheckDocument"
const dbCheckValue = http.StatusOK

// HandlerStatus Handler for the status endpoint
func HandlerStatus(cfg *util.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("content-type", "application/json")

			var countryService string
			if cfg.DevelopmentMode {
				countryService = consts.StubDomain
			} else {
				countryService = consts.CountryDomain
			}

			countriesStatus, err := util.GetDomainStatus(countryService)
			if err != nil {
				http.Error(w, "Error while handling request.", http.StatusInternalServerError)
				return
			}

			// add a known document to DB
			// TODO move to main; overwrite document at every startup?
			/*
				checkData := make(map[string]int)
				checkData["status code"] = dbCheckValue
				err = fsutils.AddDocumentById(cfg, dbCheckCollection, dbCheckDocument, checkData)
				if err != nil {
					log.Println("could not add check document to db")
				}
			*/

			// Read back document with stored status code:
			notificationStatusCode := make(map[string]int)
			err = fsutils.ReadDocumentGeneral(cfg, dbCheckCollection, dbCheckDocument, &notificationStatusCode)
			if err != nil {
				http.Error(w, "Error while handling request.", http.StatusInternalServerError)
				return
			}
			notificationStatus := fmt.Sprint(notificationStatusCode["status code"]) + " " + http.StatusText(notificationStatusCode["status code"])

			webhookCount, err := countWebhooks(cfg)
			var webhooks string
			if err != nil {
				webhooks = "Unable to count"
			} else {
				webhooks = strconv.Itoa(webhookCount)
			}

			serviceStatus := ServiceStatus{
				CountriesApi:    countriesStatus,
				NotificationsDb: notificationStatus,
				Webhooks:        webhooks,
				Version:         consts.Version,
				Uptime:          getUptime(),
			}
			// json response to user:
			util.EncodeAndWriteResponse(&w, serviceStatus)

		default:
			http.Error(w, "http method not supported.", http.StatusMethodNotAllowed)
		}
	}
}

// getUptime returns uptime since last service restart
func getUptime() int {
	return int(time.Now().Sub(StartTime).Seconds())
}

// countWebhooks returns number of stored webhooks in Firebase
func countWebhooks(cfg *util.Config) (int, error) {
	count, err := fsutils.CountDocuments(cfg, cfg.WebhookCollection)
	if err != nil {
		log.Println("could not get webhooks count")
		return 0, err
	}
	return count, nil
}
