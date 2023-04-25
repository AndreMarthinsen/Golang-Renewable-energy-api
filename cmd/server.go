package main

import (
	"Assignment2/caching"
	"Assignment2/consts"
	"Assignment2/handlers"
	"Assignment2/handlers/notifications"
	"Assignment2/internal/stubbing"
	"Assignment2/util"
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
		log.Fatal("service startup: ", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Println("main: $PORT has been set. Default: " + consts.DefaultPort)
		port = consts.DefaultPort
	}

	config, err := util.SetUpServiceConfig(consts.ConfigPath, consts.CredentialsPath)
	if err != nil {
		log.Fatal("service startup: unable to utilize firebase: ", err)
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
	serviceStartTime := time.Now()
	statusHandler := handlers.HandlerStatus(&config, serviceStartTime)

	http.HandleFunc(consts.RenewablesPath, handlers.HandlerRenew(&config, requestChannel, &countryDataset, invocation))
	http.HandleFunc(consts.NotificationPath, notificationHandler)
	http.HandleFunc(consts.StatusPath, statusHandler)

	log.Println("main: service listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
	// stub service can now be stopped with: stubStop <- struct{}{}

}
