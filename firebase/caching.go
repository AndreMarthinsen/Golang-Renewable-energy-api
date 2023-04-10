package firebase

import (
	"Assignment2/util"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type RequestStatus int16

type CacheResponse struct {
	Neighbours string
	Status     RequestStatus
}

type CacheRequest struct {
	ChannelPtr     *chan CacheResponse
	CountryRequest string
}

type CountryNeighbours struct {
}

type CountryBorder struct {
	Borders []string `json:"borders"`
}

func handleMiss(client *http.Client, method string, countries string, target interface{}) bool {
	request, err := http.NewRequest(method, consts., nil)
	if err != nil {
		log.Println("Cache Worker failed to create request.")
		return false
	}

	response, err := handler.Client.Do(request)
	if err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "request to" + request.URL.Path + " failed", err
	}

	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(target); err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "failed to decode", err
	}
	return "", nil
}

func CacheWorker(debug bool, updateFrequency float64, requests <-chan CacheRequest,
	stop <-chan struct{}, cleanupDone chan<- struct{}) {

	localCache := make(map[string]string, 0)
	cacheMisses := make([]CacheRequest, 0)
	client := http.Client{}
	context := util.HandlerContext{"Cache Worker", }

	updateDB := func() {}
	previousUpdate := time.Now()

	updateDB() //updates local cache

	for { // Runs loop until it receives signal on stop channel
		select {
		case <-stop:
			updateDB()
			cleanupDone <- struct{}{}
			return
		default:
			if time.Since(previousUpdate).Seconds() >= updateFrequency {
				updateDB()
				previousUpdate = time.Now()
			}
			for {
				select {
				case val, ok := <-requests:
					if ok { // Received a request, handles it directly on a cache hit, or defers it to the next loop
						cacheResult, ok := localCache[val.CountryRequest]
						if ok {
							*val.ChannelPtr <- CacheResponse{cacheResult, http.StatusOK}
						} else {
							cacheMisses = append(cacheMisses, val)
						}
					} else { // No connection to request channel, shuts down.
						log.Println("Cache worker lost contact with request channel.\n" +
							"Running cleanup routine and shutting down cache worker.")
						updateDB()
						cleanupDone <- struct{}{}
						return
					}
				default:
					for miss := range cacheMisses {

						util.handleMiss()
					}
				}

			}

		}
	}

}
