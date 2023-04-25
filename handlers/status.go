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

// ServiceStatus for storage of status data before encoding to json
type ServiceStatus struct {
	CountriesApi    string `json:"countries_api"`
	NotificationsDb string `json:"notification_db"`
	Webhooks        string `json:"webhooks"`
	Version         string `json:"version"`
	Uptime          int    `json:"uptime"`
}

// Collection with one document, to check if db is available:
const dbProbeCollection = "dbProbeCollection"
const dbProbeDocument = "dbProbeDocument"

// TODO remove when everything works
//const dbProbeValue = http.StatusOK

// HandlerStatus Handler for the status endpoint
func HandlerStatus(cfg *util.Config, startTime time.Time) func(http.ResponseWriter, *http.Request) {
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

			countriesStatus, err := util.GetDomainStatus2(countryService)
			if err != nil {
				log.Println("handler status: Failed to close body of get request.")
			}
			/*
				countriesStatus, err := util.GetDomainStatus(countryService)
					if err != nil {
						http.Error(w, "Error while handling request.", http.StatusInternalServerError)
						return
					}
			*/

			// Read back document with stored status code:
			notificationStatusCode := make(map[string]int)
			err = fsutils.ReadDocumentGeneral(cfg, dbProbeCollection, dbProbeDocument, &notificationStatusCode)
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
			upTime := int(time.Since(startTime).Seconds())
			serviceStatus := ServiceStatus{
				CountriesApi:    countriesStatus,
				NotificationsDb: notificationStatus,
				Webhooks:        webhooks,
				Version:         consts.Version,
				Uptime:          upTime,
			}
			// json response to user:
			util.EncodeAndWriteResponse(&w, serviceStatus)

		default:
			http.Error(w, "http method not supported.", http.StatusMethodNotAllowed)
		}
	}
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
