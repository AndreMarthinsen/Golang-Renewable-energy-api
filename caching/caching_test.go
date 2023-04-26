package caching

import (
	"Assignment2/consts"
	"Assignment2/internal/stubbing"
	"Assignment2/util"
	"context"
	"firebase.google.com/go"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestRunCacheWorker(t *testing.T) {
	requests := make(chan CacheRequest, 10)

	runCacheTest := func(codes []string, expected CacheResponse) func(*testing.T) {
		return func(t *testing.T) {
			ret := make(chan CacheResponse)
			requests <- CacheRequest{ChannelRef: ret, CountryRequest: codes}
			result := <-ret
			if result.Status != expected.Status {
				t.Error("Expected status ", expected.Status, ", got ", result.Status)
			}
			if len(result.Neighbours) != len(expected.Neighbours) {
				log.Println(result.Neighbours, expected.Neighbours)
				t.Error("Map lengths do not match.")
			}
		}
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile("../cmd/sha.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("failed to to create new app")
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal("Failed to set up caching client")
	}

	config := util.Config{
		CachePushRate:     5 * time.Second,
		CacheTimeLimit:    30 * time.Minute,
		DebugMode:         false,
		DevelopmentMode:   true,
		Ctx:               &ctx,
		FirestoreClient:   client,
		CachingCollection: "Caches",
		PrimaryCache:      "TestData",
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	wg := sync.WaitGroup{}
	defer wg.Wait()

	go stubbing.RunSTUBServer(&config, &wg, "../internal/assets/", consts.StubPort, stop)
	go RunCacheWorker(&config, requests, stop, done)

	time.Sleep(time.Second * 1)
	tests := []struct {
		name     string
		queries  []string
		expected CacheResponse
	}{
		{"test_1", []string{"NOR", "K"},
			CacheResponse{
				Status:     http.StatusOK,
				Neighbours: map[string][]string{"NOR": {"FIN", "SWE", "RUS"}},
			},
		},
		{"test_2", []string{"INVALID", "K"},
			CacheResponse{
				Status:     http.StatusNotFound,
				Neighbours: map[string][]string{},
			},
		},
		{"test_3", []string{"NOR", "KOR"},
			CacheResponse{
				Status:     http.StatusOK,
				Neighbours: map[string][]string{"NOR": {"FIN", "SWE", "RUS"}, "KOR": {"PRK"}},
			},
		},
		{"test_3", []string{"S", "KOR"},
			CacheResponse{
				Status:     http.StatusOK,
				Neighbours: map[string][]string{"KOR": {"PRK"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, runCacheTest(tt.queries, tt.expected))
	}
	time.Sleep(config.CachePushRate * 2)

}

func TestInvocationWorker(t *testing.T) {
	config, err := util.SetUpServiceConfig("../config/config.yaml", "../cmd/sha.json")
	if err != nil {
		t.Error(err)
	}

	var countryDB util.CountryDataset
	err = countryDB.Initialize("../internal/assets/renewable-share-energy.csv")
	if err != nil {
		t.Error(err)
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	invocations := make(chan []string, 10)

	go InvocationWorker(&config, stop, done, &countryDB, invocations)
	countries := []string{"NOR", "SWE", "RUS", "GER"}
	invocations <- countries
	log.Println("sleeping")
	//time.Sleep(config.WebhookEventRate)
	log.Println("done sleeping")
	stop <- struct{}{}
	log.Println("awaiting done signal")
	<-done
	log.Println("past the done signal")

}
