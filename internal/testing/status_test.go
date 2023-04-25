package testing

import (
	"Assignment2/consts"
	"Assignment2/handlers"
	"Assignment2/internal/stubbing"
	"Assignment2/util"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHandlerStats(t *testing.T) {
	config, _ := util.SetUpServiceConfig(consts.ConfigPath, "./sha.json")
	startTime := time.Now()
	time.Sleep(1 * time.Second)

	handler := handlers.HandlerStatus(&config, startTime)
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	runStatusTest := func(expected handlers.ServiceStatus) func(*testing.T) {
		return func(t *testing.T) {
			client := http.Client{}
			defer client.CloseIdleConnections()
			status := handlers.ServiceStatus{}
			url := server.URL + consts.StatusPath
			request, err := http.NewRequest(http.MethodGet, url, nil)
			response, err := client.Do(request)
			if err != nil {
				t.Error(err.Error())
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Fatal(err)
				}
			}(response.Body)
			decoder := json.NewDecoder(response.Body)
			// Error leads to a fail only if failing to decode json as a country struct is unexpected.
			if err = decoder.Decode(&status); err != nil {
				t.Error(err.Error())
			}
			if expected.CountriesApi != status.CountriesApi {
				t.Error("countries status: expected ", expected.CountriesApi,
					" got ", status.CountriesApi)
			}
			if expected.NotificationsDb != status.NotificationsDb {
				t.Error("countries firestore: expected ", expected.CountriesApi,
					" got ", status.CountriesApi)
			}
			if status.Uptime == 0 {
				t.Error("no uptime")
			}
			log.Println("done")
		}
	}
	expected := handlers.ServiceStatus{
		CountriesApi:    "503 Service Unavailable",
		NotificationsDb: "200 OK",
		Webhooks:        "",
		Version:         "",
		Uptime:          0,
	}
	t.Run("no_service_test", runStatusTest(expected))

	wg := sync.WaitGroup{}
	stop := make(chan struct{})
	wg.Add(1)
	go stubbing.RunSTUBServer(&config, &wg, consts.StubPort, stop)
	time.Sleep(time.Second)
	expected = handlers.ServiceStatus{
		CountriesApi:    "200 OK",
		NotificationsDb: "200 OK",
		Webhooks:        "",
		Version:         "",
		Uptime:          0,
	}
	t.Run("service_test", runStatusTest(expected))
	stop <- struct{}{}
	wg.Wait()
}
