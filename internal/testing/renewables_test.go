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
const historyPath = consts.RenewablesPath + "history/"
const neighbourAffix = "?neighbours=true"

// The number of tests that will be run concurrently
const concurrentTestNumber = 100

var wg sync.WaitGroup

// TestRenewables tests the renewables/ endpoint, for both current and history
func TestRenewables(t *testing.T) {
	defer wg.Wait()
	// sets up firestore context and credentials
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

	// sets up the configuration, including the firestore context and caching variables
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
	// if the program is in development mode, a stubserver is run as a goroutine
	stubStop := make(chan struct{})
	if config.DevelopmentMode {
		wg.Add(1)
		go stubbing.RunSTUBServer(&config, &wg, consts.StubPort, stubStop)
	}

	// makes 10 channels for the cacheworker
	var requestChannel = make(chan caching.CacheRequest, 10)
	stopSignal := make(chan struct{})
	doneSignal := make(chan struct{})

	// starts a goroutine for the cacheworker
	go caching.RunCacheWorker(&config, requestChannel, stopSignal, doneSignal)

	defer func() { // TODO: Just use a wait group, if that's better
		stopSignal <- struct{}{}
		<-doneSignal
	}()

	// TODO: dummy invocation channel here.
	// Invocation worker setup
	invocationStop := make(chan struct{})
	defer func() {
		invocationStop <- struct{}{}
	}()
	invocation := make(chan []string, 10)
	countryDataset, err := util.InitializeDataset(consts.DataSetPath)
	go caching.InvocationWorker(&config, invocationStop, countryDataset, invocation)

	if err != nil {
		// TODO: log an internal server error instead
		log.Print(err)
		return
	}
	// Sets handler to the renewables handler
	handler := handlers.HandlerRenew(requestChannel, countryDataset, invocation)

	server := httptest.NewServer(http.HandlerFunc(handler))
	// URL under which server is instantiated
	log.Println(server.URL)
	defer server.Close()
	client := http.Client{}

	runHandlerTest := func(wg *sync.WaitGroup, query string, expectedCode string, routine bool) func(*testing.T) {
		return func(t *testing.T) {
			// if the test has been run as part of a go-routine, it will defer signal the
			// wait group that the routine is done until the function exits/returns
			if routine {
				defer wg.Done()
			}
			statistics := make([]handlers.RenewableStatistics, 0)
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
			// if the first element of the decoded statsitcs is wrong, the test will faill
			// for situations like fetching information about all countries this might be too lenient a test
			// The alternative is to have a expected slice that encapsulates ALL information in the dataset
			if statistics[0].Isocode != expectedCode {
				t.Error("Unexpected query returned. Expected: ",
					expectedCode, " but got ", statistics[0].Isocode)
			}
		}
	}

	// the test, including the name, code to be tested and expected code in the first element of the decoded response
	var tests = []struct {
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

	// runs a number of concurrent tests equal to testnumber
	wg.Add(concurrentTestNumber)
	for i := 0; i < concurrentTestNumber; i++ {
		randomNumber := rand.Intn(8)
		go t.Run("/current test for country code "+tests[randomNumber].name+" with neighbour query",
			runHandlerTest(&wg,
				server.URL+currentPath+tests[randomNumber].query+neighbourAffix,
				tests[randomNumber].expected,
				true))
	}

	// runs test for all countries in renewables/current/ endpoint
	t.Run("All /current countries test", runHandlerTest(&wg, server.URL+currentPath, "ALG", false))

	// runs tests for random countries in historical handler
	for i := 0; i < 10; i++ {
		randomNumber := rand.Intn(8)
		t.Run("/history test for country code "+tests[randomNumber].name,
			runHandlerTest(&wg,
				server.URL+historyPath+tests[randomNumber].query+neighbourAffix,
				tests[randomNumber].expected,
				false))
	}

	// runs test for all countries in renewable/history endpoint
	t.Run("All /current countries test", runHandlerTest(&wg, server.URL+historyPath, "ALG", false))
}
