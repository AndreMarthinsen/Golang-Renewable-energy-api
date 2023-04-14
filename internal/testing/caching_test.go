package testing

import (
	"Assignment2/consts"
	caching "Assignment2/firebase"
	"Assignment2/internal/stubbing"
	"context"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestRunCacheWorker(t *testing.T) {
	requests := make(chan caching.CacheRequest, 10)

	runCacheTest := func(codes []string, expected caching.CacheResponse) func(*testing.T) {
		return func(t *testing.T) {
			ret := make(chan caching.CacheResponse)
			requests <- caching.CacheRequest{ChannelRef: ret, CountryRequest: codes}
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
	opt := option.WithCredentialsFile("./sha.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("failed to to create new app")
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal("Failed to set up firebase client")
	}

	config := caching.Config{
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

	go stubbing.RunSTUBServer(&wg, consts.StubPort)
	go caching.RunCacheWorker(&config, requests, stop, done)

	time.Sleep(time.Second * 1)
	tests := []struct {
		name     string
		queries  []string
		expected caching.CacheResponse
	}{
		{"test_1", []string{"NOR", "K"},
			caching.CacheResponse{
				Status:     http.StatusOK,
				Neighbours: map[string][]string{"NOR": {"FIN", "SWE", "RUS"}},
			},
		},
		{"test_2", []string{"INVALID", "K"},
			caching.CacheResponse{
				Status:     http.StatusNotFound,
				Neighbours: map[string][]string{},
			},
		},
		{"test_3", []string{"NOR", "KOR"},
			caching.CacheResponse{
				Status:     http.StatusOK,
				Neighbours: map[string][]string{"NOR": {"FIN", "SWE", "RUS"}, "KOR": {"PRK"}},
			},
		},
		{"test_3", []string{"S", "KOR"},
			caching.CacheResponse{
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
