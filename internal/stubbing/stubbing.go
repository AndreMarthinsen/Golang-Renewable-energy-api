// Package stubbing supplies functionality for testing handlers locally.

package stubbing

import (
	"Assignment2/consts"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

const assetPrefix = "./internal/assets/"
const codesPrefix = "codes="

// For future reference https://www.iban.com/country-codes

// parseFile parses a file specified by filename
//
// On failure: Calls log.Fatal detailing the error.
// On success: Returns the read file as a byte slice.
func parseFile(filePath string) []byte {
	file, e := os.ReadFile(filePath)
	if e != nil {
		log.Fatalf("File error: %v\n", e)
	}
	return file
}

// StubHandler simulates interacting with the third party RESTCountries API by returning appropriate
// json bodies based on input requests. Currently only simulates appropriate behaviour for the /alpha/
// endpoint using a ?codes=xxx,xxx,xxx query.
//
// debug == true: Extra information is provided in log when handler is called.
//
// Example:
// http://localhost:8888/v3.1/alpha/?codes=NOR,KOR
// Returns json file containing data for Norway and South Korea
func StubHandler(debug bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		path := r.URL.Path
		if debug {
			log.Println("Stub handler called with path " + r.URL.Path)
		}
		switch path { // Uses switch for easy expansion
		case consts.CountryCodePath:
			codes := strings.FieldsFunc(
				r.URL.Query().Get("codes"),
				func(c rune) bool { return c == ',' },
			)
			if debug {
				log.Println("stub debug: cca3 queries prior to filtering: ", codes)
			}
			codes = filterCountryCodes(codes)
			if len(codes) == 0 { // Indicates no codes of valid length [2, 3]
				response := "{\"status\":400,\"message\":\"Bad Request\"}"
				if _, err := fmt.Fprint(w, response); err != nil {
					log.Fatal("stub handler failed to return response body to client.")
				}
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			response, err := getJsonByCountryCode(codes)
			if err != nil {
				response = "{\"status\":404,\"message\":\"Not Found\"}"
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			if _, err = fmt.Fprint(w, response); err != nil {
				log.Fatal("stub handler failed to return response body to client.")
			}
			return
		default:
			if debug {
				log.Println("Path: " + r.URL.Path + " not currently supported by stubbing service.")
			}
			http.Error(w, "Not a recognized path for stubbing", http.StatusNotImplemented)
		}
	}
}

// getJsonByCountryCode takes a slice of country codes, returning all results
// for those country codes.
// WARNING: For any simulated response there must be a .json file in the /internal/assets directory.
// For the simulation of invalid requests, use an empty .json file, such as codes=INV.json
// Attempting to read a non-existing file will intentionally crash the application.
func getJsonByCountryCode(countryCodes []string) (string, error) {
	countryData := make([]string, 0)
	for _, code := range countryCodes {
		data := string(parseFile(assetPrefix + codesPrefix + code + ".json"))
		if len(data) >= 2 {
			data = strings.TrimPrefix(strings.TrimSuffix(data, "]"), "[")
			countryData = append(countryData, data)
		}
	}
	if len(countryData) == 0 {
		return "", errors.New("failed to find any match on provided country codes")
	}
	return "[" + strings.Join(countryData, ",") + "]", nil
}

// filterCountryCodes filters out any code that is not 2 or 3 characters long as these result
// in being ignored by RESTCountries if sent along with other countries, or resulting in a
// 400 statusBadRequest if the only code.
func filterCountryCodes(countryCodes []string) []string {
	filteredCodes := make([]string, 0)
	for _, code := range countryCodes {
		if len(code) == 2 || len(code) == 3 {
			filteredCodes = append(filteredCodes, code)
		}
	}
	return filteredCodes
}

// RunSTUBServer runs a stubbing service using the net/http module.
// See StubHandler for closer detail on what stubbing is provided by the service.
func RunSTUBServer(group *sync.WaitGroup, port string, stop chan struct{}) {
	defer group.Done()

	log.Println("STUB service running on port", port)

	server := http.Server{
		Addr:    ":" + port,
		Handler: http.HandlerFunc(StubHandler(true)),
	}

	go func() {
		err := server.ListenAndServe()
		log.Println("stub service shut down: ", err)
	}()

	<-stop // waits on stop signal to shut down the stub server
	if err := server.Shutdown(nil); err != nil {
		log.Println("failed to properly shut down stubbing service")
	}

}
