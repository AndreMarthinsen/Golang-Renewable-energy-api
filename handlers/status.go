package handlers

import (
	"Assignment2/consts"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// StartTime for calculating service uptime
var StartTime = time.Now()

// ServiceStatus for storage of status data before encoding to json
type ServiceStatus struct {
	CountriesApi int    `json:"countries_api"`
	EnergyApi    int    `json:"notification_db"`
	Webhooks     int    `json:"webhooks"`
	Version      string `json:"version"`
	Uptime       int    `json:"uptime"`
}

// HandlerStatus Handler for the status endpoint
func HandlerStatus(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("content-type", "application/json")

		serviceStatus := ServiceStatus{
			CountriesApi: statusClient(consts.CountriesUrl),
			EnergyApi:    statusClient(consts.EnergyUrl),
			Webhooks:     countWebhooks(),
			Version:      consts.Version,
			Uptime:       getUptime(),
		}
		// json response to user:
		encoder := json.NewEncoder(w)
		err := encoder.Encode(serviceStatus)
		if err != nil {
			http.Error(w, "Error while encoding to json", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "http method not supported.", http.StatusMethodNotAllowed)
	}

}

// statusClient sends requests to 3rd party services
func statusClient(url string) (status int) {
	status = http.StatusInternalServerError // default to 500

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("Error while creating status request.")
		return status
	}

	// Client instantiation:
	client := http.Client{}
	defer client.CloseIdleConnections()

	// Request/response:
	res, err := client.Do(req)
	if err != nil {
		log.Println("Unable to send request.")
		return status
	}

	status = res.StatusCode
	return status
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
