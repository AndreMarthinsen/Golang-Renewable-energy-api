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

var wg sync.WaitGroup

// TestRenewables tests the renewables/ endpoint, for both current and history
func TestRenewables(t *testing.T) {
	// Setup of firebase context and application
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

	// Initialization of dataset from CSV
	var countryDataset util.CountryDataset
	err = countryDataset.Initialize(consts.DataSetPath)
	if err != nil {
		log.Fatal(err)
	}

	// sets up the server configuration
	config := util.Config{
		CachePushRate:     5 * time.Second,
		CacheTimeLimit:    2 * time.Hour,
		DebugMode:         false,
		DevelopmentMode:   true,
		Ctx:               &ctx,
		FirestoreClient:   firestoreClient,
		CachingCollection: "Caches",
		PrimaryCache:      "TestData",
		WebhookCollection: "Webhooks",
	}

	// Setup of communication channels used with worker threads
	requests := make(chan caching.CacheRequest, 10)
	stubStop := make(chan struct{})
	cacheStop := make(chan struct{})
	cacheDone := make(chan struct{})
	invocations := make(chan []string, 10)
	invocationStop := make(chan struct{})

	t.Cleanup(func() {
		cacheStop <- struct{}{}
		stubStop <- struct{}{}
		invocationStop <- struct{}{}
	})
	// Launch of worker threads
	if config.DevelopmentMode {
		wg.Add(1)
		go stubbing.RunSTUBServer(&config, &wg, consts.StubPort, stubStop)
	}
	go caching.RunCacheWorker(&config, requests, cacheStop, cacheDone)
	go caching.InvocationWorker(&config, invocationStop, &countryDataset, invocations)

	// Injection of dependencies into the handler
	testHandler := handlers.HandlerRenew(&config, requests, &countryDataset, invocations)

	runHandlerTest := func(wg *sync.WaitGroup, query string, expectedCode string, routine bool, expectedLength int) func(*testing.T) {
		return func(t *testing.T) {
			// if the test has been run as part of a go-routine, it will defer signal the
			// wait group that the routine is done until the function exits/returns
			if routine {
				t.Parallel()
			}
			server := httptest.NewServer(http.HandlerFunc(testHandler))
			query = server.URL + query
			client := http.Client{}

			defer client.CloseIdleConnections()
			defer server.Close()

			statistics := make([]util.RenewableStatistics, 0)
			request, err := http.NewRequest(http.MethodGet, query, nil)
			if err != nil {
				t.Error(err.Error())
			}
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
			if expectedCode == "" {
				if len(statistics) != expectedLength {
					t.Error("Unexpected length returned. Expected: ",
						expectedLength, " but got ", len(statistics))
				}
			} else {
				// if the first element of the decoded statsitcs is wrong, the test will faill
				// for situations like fetching information about all countries this might be too lenient a test
				// The alternative is to have an expected slice that encapsulates ALL information in the dataset
				if len(statistics) != 0 && statistics[0].Isocode != expectedCode {
					t.Error("Unexpected query returned. Expected: ",
						expectedCode, " but got ", statistics[0].Isocode)
				}
			}
		}
	}

	// the test, including the name, code to be tested and expected code in the first element of the decoded response
	var tests = []struct {
		name       string
		query      string
		expected   string
		neighbours int
	}{
		{"CHN test", "CHN", "CHN", 0},
		{"FIN test", "FIN", "FIN", 0},
		{"KOR test", "KOR", "KOR", 0},
		{"NOR test", "NOR", "NOR", 0},
		//{"PRK test", "PRK", "PRK"},
		{"RUS test", "RUS", "RUS", 0},
		{"SWE test", "SWE", "SWE", 0},
		//{"TJK test", "TJK", "TJK"},
		{"UZB test", "UZB", "UZB", 0},
		{"VNM test", "VNM", "VNM", 0},
	}

	// runs tests for random countries in historical testHandler
	for i := 0; i < 10; i++ {
		randomNumber := rand.Intn(8)
		t.Run("/history test for country code "+tests[randomNumber].name,
			runHandlerTest(&wg,
				historyPath+tests[randomNumber].query,
				tests[randomNumber].expected,
				true, 1))
	}

	err, datasetLength := countryDataset.GetLengthOfDataset()
	if err != nil {
		t.Error(err)
	}
	// runs test for all countries in renewable/history endpoint
	t.Run("All /current countries test", runHandlerTest(&wg, currentPath, "", true, datasetLength))

	// runs a number of concurrent tests equal to testnumber
	for i := 0; i < 100; i++ {
		randomNumber := rand.Intn(8)
		t.Run("/current test for country code "+tests[randomNumber].name+" with neighbour query",
			runHandlerTest(&wg,
				currentPath+tests[randomNumber].query+neighbourAffix,
				tests[randomNumber].expected,
				true, 0))
	}
}
