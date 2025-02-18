package handlers

import (
	"Assignment2/caching"
	"Assignment2/consts"
	"Assignment2/internal/stubbing"
	"Assignment2/util"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Internal paths
const currentTestPath = consts.RenewablesPath + "current/"
const historyTestPath = consts.RenewablesPath + "history/"
const neighbourAffix = "?neighbours=true"

// TestRenewables tests the renewables/ endpoint, for both current and history
func TestRenewables(t *testing.T) {
	wg := sync.WaitGroup{}

	// Initialization of dataset from CSV
	var countryDataset util.CountryDataset
	err := countryDataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		log.Fatal(err)
	}
	// sets up the server configuration
	config, err := util.SetUpServiceConfig("."+consts.ConfigPath, "../cmd/sha.json")
	if err != nil {
		log.Fatal("service startup: unable to utilize firebase: ", err)
	}

	// Setup of communication channels used with worker threads
	requests := make(chan caching.CacheRequest, 10)
	stubStop := make(chan struct{})
	cacheStop := make(chan struct{})
	cacheDone := make(chan struct{})
	invocations := make(chan []string, 10)
	invocationStop := make(chan struct{})
	invocationDone := make(chan struct{})

	t.Cleanup(func() {
		cacheStop <- struct{}{}
		stubStop <- struct{}{}
		invocationStop <- struct{}{}
		<-cacheDone
		<-invocationDone
	})
	// Launch of worker threads
	if config.DevelopmentMode {
		wg.Add(1)
		go stubbing.RunSTUBServer(&config, &wg, "../internal/assets/", consts.StubPort, stubStop)
	}
	go caching.RunCacheWorker(&config, requests, cacheStop, cacheDone)
	go caching.InvocationWorker(&config, invocationStop, invocationDone, &countryDataset, invocations)

	// Injection of dependencies into the handler
	testHandler := HandlerRenew(requests, &countryDataset, invocations)

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
				// if no expected code has been sent, it will instead check if response has the correct length
				if len(statistics) != expectedLength {
					t.Error("Unexpected length returned. Expected: ",
						expectedLength, " but got ", len(statistics))
				}
			} else {
				// if the first element of the decoded statistics is wrong, the test will fail
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
		country    string
		query      string
		expected   string
		neighbours int
	}{
		{"CHN test", "china", "CHN", "CHN", 6},
		{"FIN test", "finland", "FIN", "FIN", 3},
		{"KOR test", "south korea", "KOR", "KOR", 0},

		{"NOR test", "norway", "NOR", "NOR", 3},
		{"RUS test", "russia", "RUS", "RUS", 11},
		{"SWE test", "sweden", "SWE", "SWE", 2},

		{"UZB test", "uzbekistan", "UZB", "UZB", 2},
		{"VNM test", "vietnam", "VNM", "VNM", 1},
		{"Invalid test", "", "INV", "", 0},
	}

	err, datasetLength := countryDataset.GetLengthOfDataset()
	if err != nil {
		t.Error(err)
	}

	// runs tests for random countries in historical testHandler
	for i := 0; i < 5; i++ {
		randomNumber := rand.Intn(8)
		t.Run("/history test for country code "+tests[randomNumber].name,
			runHandlerTest(&wg,
				historyTestPath+tests[randomNumber].query,
				tests[randomNumber].expected,
				true, 1))
	}

	// runs tests for random countries in historical endpoint, querying by name
	for i := 0; i < 5; i++ {
		randomNumber := rand.Intn(8)
		t.Run("/history test for country name "+tests[randomNumber].name,
			runHandlerTest(&wg,
				historyTestPath+tests[randomNumber].country,
				tests[randomNumber].expected,
				true, 1))
	}

	// runs test for all countries in renewable/current endpoint
	t.Run("All /current countries test",
		runHandlerTest(&wg,
			currentTestPath,
			"",
			true, datasetLength))
	// runs test for all countries in renewable/history endpoint
	t.Run("All /history countries test",
		runHandlerTest(&wg,
			historyTestPath,
			"",
			true, datasetLength))
	// runs test for all countries in renewable/history endpoint
	t.Run("All /history countries test, sorted by value",
		runHandlerTest(&wg,
			historyTestPath+"?sortByValue=true",
			"SAU",
			true, datasetLength))
	// runs test for all countries in renewable /history endpoint, with year limitation
	t.Run("All /history countries test, between 1995 and 2006",
		runHandlerTest(&wg,
			historyTestPath+"?begin=1995&end=2006",
			"",
			true, datasetLength))

	// runs a number of tests for current endpoint with neighbour query
	for i := 0; i < 5; i++ {
		randomNumber := rand.Intn(8)
		t.Run("/current test for country code "+tests[randomNumber].name+" with neighbour query",
			runHandlerTest(&wg,
				currentTestPath+tests[randomNumber].query+neighbourAffix,
				tests[randomNumber].expected,
				true, 1+tests[randomNumber].neighbours))
	}

	// runs a number of tests for current endpoint without neighbour query
	for i := 0; i < 5; i++ {
		randomNumber := rand.Intn(8)
		t.Run("current test for country code "+tests[randomNumber].name,
			runHandlerTest(&wg,
				currentTestPath+tests[randomNumber].query,
				tests[randomNumber].expected,
				true, 1))
	}

	// runs tests for random countries in historical endpoint, querying by name
	for i := 0; i < 5; i++ {
		randomNumber := rand.Intn(8)
		t.Run("/current test for country name "+tests[randomNumber].name,
			runHandlerTest(&wg,
				currentTestPath+tests[randomNumber].country,
				tests[randomNumber].expected,
				true, 1))
	}

	// runs a test for beginning year query in history endpoint
	t.Run("/history test for "+tests[0].name+" with begin query",
		runHandlerTest(&wg,
			historyTestPath+tests[0].query+"?begin=2000",
			tests[0].expected,
			true, 1))

	// runs an invalid test for beginning year query in history endpoint
	t.Run("/history test for "+tests[0].name+" with begin query",
		runHandlerTest(&wg,
			historyTestPath+tests[0].query+"?begin=sljh",
			"",
			true, 0))

	// runs a test for end year query in history endpoint
	t.Run("/history test for "+tests[1].name+" with end query",
		runHandlerTest(&wg,
			historyTestPath+tests[1].query+"?end=2000",
			tests[1].expected,
			true, 1))

	// runs a invalid test for end year query in history endpoint
	t.Run("/history test for "+tests[2].name+" with end query",
		runHandlerTest(&wg,
			historyTestPath+tests[2].query+"?end=erte",
			"",
			true, 0))

	// runs an invalid query for current endpoint
	t.Run(tests[8].name+" for current endpoint",
		runHandlerTest(&wg,
			currentTestPath+tests[8].query,
			tests[8].expected,
			true,
			0))

	// runs an invalid query for history endpoint
	t.Run(tests[8].name+" for history endpoint",
		runHandlerTest(&wg,
			historyTestPath+tests[8].query,
			tests[8].expected,
			true,
			0))
}

func TestSortSlice(t *testing.T) {
	runTest := func(stats []util.RenewableStatistics, firstExpected util.RenewableStatistics) func(t *testing.T) {
		return func(t *testing.T) {
			stats = SortStatistics(stats)
			if stats[0] != firstExpected {
				t.Errorf("Unexpected sorted result, got #{stats[0]} but expected #{firstExpected}")
			}
		}
	}

	testCases := []struct {
		name          string
		statistics    []util.RenewableStatistics
		firstExpected util.RenewableStatistics
	}{
		{name: "Test for 1960s Norway statiscs",
			statistics: []util.RenewableStatistics{
				{"Norway", "NOR", 1967, 60.32},
				{"Norway", "NOR", 1968, 61.132},
				{"Norway", "NOR", 1969, 62.31},
			},
			firstExpected: util.RenewableStatistics{"Norway", "NOR", 1967, 60.32}},
		{name: "Test for 1970s Norway statiscs",
			statistics: []util.RenewableStatistics{
				{"Norway", "NOR", 1972, 59.81},
				{"Norway", "NOR", 1974, 58.82},
				{"Norway", "NOR", 1978, 62.01},
			},
			firstExpected: util.RenewableStatistics{"Norway", "NOR", 1974, 58.82}},
		{name: "Test for 1990s Sweden statiscs",
			statistics: []util.RenewableStatistics{
				{"Sweden", "SWE", 1994, 48.15},
				{"Sweden", "SWE", 1996, 50.12},
				{"Sweden", "SWE", 1998, 47.01},
			},
			firstExpected: util.RenewableStatistics{"Sweden", "SWE", 1998, 47.01}},
	}
	for _, test := range testCases {
		t.Run(test.name, runTest(test.statistics, test.firstExpected))
	}
}
