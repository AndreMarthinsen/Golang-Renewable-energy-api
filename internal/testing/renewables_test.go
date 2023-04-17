package testing

import (
	"Assignment2/consts"
	"Assignment2/handlers"
	"Assignment2/caching"
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
	var requestChannel = make(chan caching.CacheRequest)
	// Sets handler to the renewables handler
	handler := handlers.HandlerRenew(requestChannel)

	server := httptest.NewServer(http.HandlerFunc(handler))
	// URL under which server is instantiated
	fmt.Println("Server running on URL: ", server.URL)

	defer server.Close()
	

	client := http.Client{}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{name: "Test 1", query: "NOR", expected: "NOR"},
		{name: "Test 2", query: "INV", expected: ""},
		{name: "Test 3", query: "NOR", expected: "NOR"},
		{name: "Test 4", query: "NOR", expected: "NOR"},
	}

	for _, tt := range tests {
		url := server.URL + currentPath + tt.query

		// Retrieve content from server
		res, err := client.Get(url)
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

		if statistics[0].Isocode != tt.expected {
			t.Fatal("First country information is wrong")
		}
	}
	
}
