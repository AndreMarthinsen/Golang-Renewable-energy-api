package main

import (
	"Assignment2/caching"
	"Assignment2/consts"
	"Assignment2/handlers"
	"Assignment2/handlers/notifications"
	"Assignment2/internal/stubbing"
	"Assignment2/util"
	"context"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var wg sync.WaitGroup

func main() {
	defer wg.Wait()
	var countryDataset util.CountryDataset
	err := countryDataset.Initialize(consts.DataSetPath)
	if err != nil {
		// TODO: log an internal server error instead
		log.Fatal(err)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Println("main: $PORT has been set. Default: " + consts.DefaultPort)
		port = consts.DefaultPort
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile("./cmd/sha.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatal("main: failed to to create new app", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal("main: failed to set up firestore client:", err)
	}

	config := util.Config{
		CachePushRate:     5 * time.Second,
		CacheTimeLimit:    2 * time.Hour,
		DebugMode:         true,
		DevelopmentMode:   true,
		Ctx:               &ctx,
		FirestoreClient:   client,
		CachingCollection: "Caches",
		PrimaryCache:      "TestData",
		WebhookCollection: "Webhooks",
	}

	// Stub server setup
	stubStop := make(chan struct{})
	if config.DevelopmentMode {
		wg.Add(1)
		go stubbing.RunSTUBServer(&config, &wg, consts.StubPort, stubStop)
	}

	// Invocation worker setup
	invocation := make(chan []string, 10)
	invocationStop := make(chan struct{})
	go caching.InvocationWorker(&config, invocationStop, &countryDataset, invocation)

	// Cache worker setup
	requestChannel := make(chan caching.CacheRequest, 10)
	stopSignal := make(chan struct{})
	doneSignal := make(chan struct{})

	go caching.RunCacheWorker(&config, requestChannel, stopSignal, doneSignal)

	defer func() { // TODO: Just use a wait group, if that's better
		stopSignal <- struct{}{}
		<-doneSignal
	}()
	notificationHandler := notifications.HandlerNotification(&config)
	statusHandler := handlers.HandlerStatus(&config)

	http.HandleFunc(consts.RenewablesPath, handlers.HandlerRenew(&config, requestChannel, &countryDataset, invocation))
	http.HandleFunc(consts.NotificationPath, notificationHandler)
	http.HandleFunc(consts.StatusPath, statusHandler)

	log.Println("main: service listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
	// stub service can now be stopped with: stubStop <- struct{}{}

}
