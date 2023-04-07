package handlers

import (
	"Assignment2/consts"
	"Assignment2/util"
	"net/http"
	"time"
)

// StartTime for calculating service uptime
var StartTime = time.Now()

// ServiceStatus for storage of status data before encoding to json
type ServiceStatus struct {
	CountriesApi string `json:"countries_api"`
	EnergyApi    string `json:"notification_db"`
	Webhooks     int    `json:"webhooks"`
	Version      string `json:"version"`
	Uptime       int    `json:"uptime"`
}

// HandlerStatus Handler for the status endpoint
func HandlerStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("content-type", "application/json")

		countries, err := util.GetDomainStatus(consts.CountriesUrl)
		if err != nil {
			http.Error(w, "Error while handling request.", http.StatusInternalServerError)
			return
		}
		energy, err := util.GetDomainStatus(consts.EnergyUrl)
		if err != nil {
			http.Error(w, "Error while handling request.", http.StatusInternalServerError)
			return
		}
		serviceStatus := ServiceStatus{
			CountriesApi: countries,
			EnergyApi:    energy,
			Webhooks:     countWebhooks(),
			Version:      consts.Version,
			Uptime:       getUptime(),
		}
		// json response to user:
		util.EncodeAndWriteResponse(&w, serviceStatus)

	default:
		http.Error(w, "http method not supported.", http.StatusMethodNotAllowed)
	}

}

// getUptime returns uptime since last service restart
func getUptime() int {
	return int(time.Now().Sub(StartTime).Seconds())
}

// countWebhooks returns number of stored webhooks in Firebase
func countWebhooks() int {
	return 0
	// TODO implement body
}
