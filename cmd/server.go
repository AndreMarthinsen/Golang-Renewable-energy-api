package main

import (
	"Assignment2/consts"
	"Assignment2/handlers"
	"Assignment2/internal/stubbing"
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
	stubStop := make(chan struct{})
	if consts.Development { // WARNING: Ensure Development is set false for release.
		wg.Add(1)
		go stubbing.RunSTUBServer(&wg, consts.StubPort, stubStop)
	}

	http.HandleFunc(consts.RenewablesPath, handlers.HandlerRenew)
	http.HandleFunc(consts.NotificationPath, handlers.HandlerNotification)
	http.HandleFunc(consts.StatusPath, handlers.HandlerStatus)

	log.Println("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
	// stub service can now be stopped with: stubStop <- struct{}{}

}
