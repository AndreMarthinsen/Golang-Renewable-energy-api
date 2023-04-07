package main

import (
	"Assignment2/consts"
	"Assignment2/handlers"
	"Assignment2/internal/stubbing"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("$PORT has been set. Default: " + consts.DefaultPort)
		port = consts.DefaultPort
	}

	codes := []string{"NOR", "SWE", "KOR"}
	fmt.Println(stubbing.GetJsonByCountryCode(codes))

	http.HandleFunc(consts.RenewablesPath, handlers.HandlerRenew)
	http.HandleFunc(consts.NotificationPath, handlers.HandlerNotification)
	http.HandleFunc(consts.StatusPath, handlers.HandlerStatus)

	log.Println("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
