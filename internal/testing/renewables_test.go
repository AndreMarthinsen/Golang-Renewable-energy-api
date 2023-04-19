package testing

import (
	"Assignment2/caching"
	"Assignment2/consts"
	"Assignment2/handlers"
	"Assignment2/internal/stubbing"
	"Assignment2/util"
	"context"
	"encoding/json"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Internal paths
const currentPath = consts.RenewablesPath + "current/"
const neighbourAffix = "?neighbours=true"

var wg sync.WaitGroup

// const historyPath = consts.RenewablesPath + "history/"
// TestCurrentRenewables tests the renewables/current/ endpoint
func TestCurrentRenewables(t *testing.T) {
	defer wg.Wait()

	ctx := context.Background()
	opt := option.WithCredentialsFile("./sha.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("failed to to create new app")
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal("Failed to set up caching client")
	}

	config := util.Config{
		CachePushRate:     5 * time.Second,
		CacheTimeLimit:    2 * time.Hour,
		DebugMode:         true,
		DevelopmentMode:   true,
		Ctx:               &ctx,
		FirestoreClient:   firestoreClient,
		CachingCollection: "Caches",
		PrimaryCache:      "TestData",
		WebhookCollection: "Webhooks",
	}
	stubStop := make(chan struct{})
	if config.DevelopmentMode {
		wg.Add(1)
		go stubbing.RunSTUBServer(&config, &wg, consts.StubPort, stubStop)
	}

	var requestChannel = make(chan caching.CacheRequest, 10)
	stopSignal := make(chan struct{})
	doneSignal := make(chan struct{})

	go caching.RunCacheWorker(&config, requestChannel, stopSignal, doneSignal)

	defer func() { // TODO: Just use a wait group, if that's better
		stopSignal <- struct{}{}
		<-doneSignal
	}()

	// Sets handler to the renewables handler
	handler := handlers.HandlerRenew(requestChannel)

	server := httptest.NewServer(http.HandlerFunc(handler))
	// URL under which server is instantiated
	log.Println(server.URL)
	defer server.Close()
	client := http.Client{}

	runCurrentHandlerTest2 := func(wg *sync.WaitGroup, query string, expectedCode string) func(*testing.T) {
		return func(t *testing.T) {
			defer wg.Done()
			statistics := make([]handlers.RenewableStatistics, 0)
			//url := server.URL + currentPath + query + neighbourAffix
			request, err := http.NewRequest(http.MethodGet, query, nil)
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
			if err = decoder.Decode(&statistics); err != nil && len(expectedCode) != 0 {
				t.Error("Get request to URL failed:", err.Error())
			}
			if statistics[0].Isocode != expectedCode {
				t.Error("Unexpected query returned. Expected: ",
					expectedCode, " but got ", statistics[0].Isocode)
			}
		}
	}

	var tt = []struct {
		name     string
		query    string
		expected string
	}{
		{"CHN test", "CHN", "CHN"},
		{"FIN test", "FIN", "FIN"},
		{"KOR test", "KOR", "KOR"},
		{"NOR test", "NOR", "NOR"},
		//{"PRK test", "PRK", "PRK"},
		{"RUS test", "RUS", "RUS"},
		{"SWE test", "SWE", "SWE"},
		//{"TJK test", "TJK", "TJK"},
		{"UZB test", "UZB", "UZB"},
		{"VNM test", "VNM", "VNM"},
	}

	wg.Add(10)
	for i := 0; i < 10; i++ {
		randomNumber := rand.Intn(8)
		go t.Run(tt[randomNumber].name+"neighbour", runCurrentHandlerTest2(&wg,
			server.URL+currentPath+tt[randomNumber].query+neighbourAffix,
			tt[randomNumber].expected))
	}

	t.Run(tt[3].name,
		runCurrentHandlerTest2(&wg, server.URL+currentPath+tt[3].query, tt[3].expected))
}
