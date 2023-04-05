package handlers

import (
	"Assignment2/consts"
	"encoding/json"
	"fmt"
	"net/http"
)

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
	w.Header().Set("content-type", "application/json")

	serviceStatus := ServiceStatus{
		CountriesApi: StatusClient(consts.CountriesUrl),
		EnergyApi:    StatusClient(consts.EnergyUrl),
		Webhooks:     countWebHooks(),
		Version:      consts.Version,
		Uptime:       getUptime(),
	}
	// Response to user:
	encoder := json.NewEncoder(w)
	err := encoder.Encode(serviceStatus)
	if err != nil {
		http.Error(w, "Error while encoding to json", http.StatusInternalServerError)
	}

}

// StatusClient Sends requests to 3rd party services
func StatusClient(url string) (status int) {
	status = http.StatusInternalServerError

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Error while creating status request.")
		return status
	}

	// Client instantiation:
	client := http.Client{}
	defer client.CloseIdleConnections()

	// Request/response:
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Unable to send request.")
		return status
	}

	status = res.StatusCode
	return status
}

// getUptime returns the uptime since last service restart
func getUptime() int {
	return 0
}

// countWebhooks returns the number of stored webhooks in Firebase
func countWebHooks() int {
	return 0
}
