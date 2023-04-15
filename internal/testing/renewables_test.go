package testing

import (
	"Assignment2/consts"
	"Assignment2/handlers"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Internal paths
const currentPath = consts.RenewablesPath + "current/"

// const historyPath = consts.RenewablesPath + "history/"
// TestCurrentRenewables tests the renewables/current/ endpoint
func TestCurrentRenewables(t *testing.T) {
	// Sets handler to the renewables handler
	handler := handlers.HandlerRenew()

	server := httptest.NewServer(http.HandlerFunc(handler))

	defer server.Close()

	client := http.Client{}

	// URL under which server is instantiated, with path for current renewables
	fmt.Println("Server running on URL: ", server.URL)

	// Retrieve content from server
	res, err := client.Get(server.URL + currentPath)
	if err != nil {
		t.Fatal("Get request to URL failed:", err.Error())
	}

	// decodes information from request
	var statistics []handlers.RenewableStatistics
	err = json.NewDecoder(res.Body).Decode(&statistics)
	if err != nil {
		t.Fatal("Error during decoding:", err.Error())
	}

	fmt.Println(len(statistics))

	if statistics[0].Name != "Algeria" {
		t.Fatal("First country information is wrong")
	}
}
