package main

import (
	"Assignment2/consts"
	caching "Assignment2/firebase"
	"Assignment2/handlers"
	"Assignment2/internal/stubbing"
	"context"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
	"sync"
)

var wg sync.WaitGroup

func main() {
	defer wg.Wait()

	port := os.Getenv("PORT")
	if port == "" {
		log.Println("$PORT has been set. Default: " + consts.DefaultPort)
		port = consts.DefaultPort
	}

	if consts.Development { // WARNING: Ensure Development is set false for release.
		wg.Add(1)
		go stubbing.RunSTUBServer(&wg, consts.StubPort)
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile("./cmd/sha.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("failed to to create new app")
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal("Failed to set up firebase client")
	}

	config := caching.Config{
		CachePushRate:     5.0,
		DebugMode:         true,
		DevelopmentMode:   true,
		Ctx:               &ctx,
		FirestoreClient:   client,
		CachingCollection: "Caches",
		PrimaryCache:      "TestData",
	}

	requestChannel := make(chan caching.CacheRequest)
	stopSignal := make(chan struct{})
	doneSignal := make(chan struct{})

	go caching.RunCacheWorker(&config, requestChannel, stopSignal, doneSignal)

	defer func() { // TODO: Just use a wait group, if that's better
		stopSignal <- struct{}{}
		<-doneSignal
	}()

	http.HandleFunc(consts.RenewablesPath, handlers.HandlerRenew)
	http.HandleFunc(consts.NotificationPath, handlers.HandlerNotification)
	http.HandleFunc(consts.StatusPath, handlers.HandlerStatus)

	log.Println("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
