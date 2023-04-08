package testing

import (
	"Assignment2/consts"
	"Assignment2/internal/stubbing"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Country struct for testing purposes. Add fields as seen fit, but do not remove any
// fields already present.
type country struct {
	Cca3 string `json:"cca3"`
	Name struct {
		Common string `json:"common"`
	} `json:"name"`
	Languages map[string]string `json:"languages"`
	Map       struct {
		OpenStreetMaps string `json:"openStreetMaps"`
	} `json:"maps"`
	Borders []string `json:"borders"`
}

func TestHttpStubbing(t *testing.T) {
	handler := stubbing.StubHandler(false)
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client := http.Client{}

	fmt.Println("http test server running with url:" + server.URL)

	// function binding for repeated testing
	// expectedCount: number of structs decoded from returned json body.
	// set expectedCount to 0 if you expect the service to not return any valid country data.
	runStubHandlerTest := func(countryCodes []string, expectedCodes []string) func(*testing.T) {
		return func(t *testing.T) {
			countries := make([]country, 0)
			url := server.URL + consts.CountryCodePath + "?codes=" + strings.Join(countryCodes, ",")
			request, err := http.NewRequest(http.MethodGet, url, nil)
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
			if err = decoder.Decode(&countries); err != nil && len(expectedCodes) != 0 {
				t.Error("Get request to URL failed:", err.Error())
			}
			returnedCodes := make([]string, 0)
			for _, code := range countries {
				returnedCodes = append(returnedCodes, code.Cca3)
			}
			if len(countries) != len(expectedCodes) {
				t.Error("Unexpected codes in returned list. Expected:",
					expectedCodes, ", but got", returnedCodes)
			}
			for i, code := range returnedCodes {
				if expectedCodes[i] != code {
					t.Error("Unexpected codes in returned list. Expected:",
						expectedCodes, ", but got", returnedCodes)
				}
			}
		}

	}

	tests := []struct {
		name     string
		queries  []string
		expected []string
	}{
		{"Test 1", []string{"NOR", "K"}, []string{"NOR"}},
		{"Test 2", []string{"NOR", "KOR"}, []string{"NOR", "KOR"}},
		{"Test 3", []string{"NOR", "INV"}, []string{"NOR"}},
		{"Test 4", []string{"SWE", "NOR", "RUS"}, []string{"SWE", "NOR", "RUS"}},
		{"Test 5", []string{"INV"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, runStubHandlerTest(tt.queries, tt.expected))
	}
}
