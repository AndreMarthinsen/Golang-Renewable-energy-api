package handlers

import (
	"fmt"
	"net/http"
)

type ServiceStatus struct {
	CountriesApi int    `json:"countries_api"`
	EnergyApi    int    `json:"notification_db"`
	Webhooks     int    `json:"webhooks"`
	Version      string `json:"version"`
	Uptime       int    `json:"uptime"`
}

// HandlerStatus Handler for the status endpoint
func HandlerStatus(w http.ResponseWriter, r *http.Request) {

}

// StatusClient Sends requests to 3rd party services
func StatusClient(url string) (status int, err error) {
	status = http.StatusServiceUnavailable

	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("Error while creating status request.")
		return status, err
	}

	// Client instantiation:
	client := http.Client{}
	defer client.CloseIdleConnections()

	// Request/response:
	res, err := client.Do(r)
	if err != nil {
		fmt.Println("Unable to send request.")
		return status, err
	}

	status = res.StatusCode
	return status, nil
}
