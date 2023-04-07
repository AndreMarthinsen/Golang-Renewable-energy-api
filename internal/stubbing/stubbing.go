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

// For futurue reference https://www.iban.com/country-codes
const cc3aVietNam = "VNM"
const cc3aNorthKorea = "PRK"
const cc3aSouthKorea = "KOR"
const cc3aNorway = "NOR"
const cc3aSweden = "SWE"
const cc3aFinland = "FIN"
const cc3aRussia = "RUS"

const cc3aTajikistan = "TJK" // no energy data
const cc3aChina = "CHN"      // neighbour of Tajikistan, does exist
const cc3aUzbekistan = "UZB" // same as above

// parseFile parses a file specified by filename
// On failure: Calls log.Fatal detailing the error.
// On success: Returns the read file as a byte slice.
func parseFile(filePath string) []byte {
	file, e := os.ReadFile(filePath)
	if e != nil {
		log.Fatalf("File error: %v\n", e)
	}
	return file
}

func StubHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch path {
	case consts.CountryCodePath:
		{
			codes := strings.FieldsFunc(
				r.URL.Query().Get("codes"),
				func(c rune) bool { return c == ',' },
			)
			response, err := GetJsonByCountryCode(codes)
			if err != nil {
				http.Error(w, "[]", http.StatusNotFound)
				return
			}
			fmt.Println(w, response)
			return
		}
	default:
		{
			//TODO: 404 message here?
		}
	}
}

// GetJsonByCountryCode takes a slice of country codes, returning all results
// for those country codes
func GetJsonByCountryCode(countryCodes []string) (string, error) {
	countryData := make([]string, 0)
	for _, code := range countryCodes {
		data := string(parseFile(assetPrefix + codesPrefix + code + ".json"))
		if len(data) >= 2 {
			data = strings.TrimPrefix(strings.TrimSuffix(data, "]"), "[")
		}
		countryData = append(countryData, data)
	}
	if len(countryData) == 0 {
		return "", errors.New("failed to find any match on provided country codes")
	}
	return "[" + strings.Join(countryData, ",") + "]", nil
}

func stubHandlerUniversities(w http.ResponseWriter, r *http.Request) {
	fmt.Println("UniStubHandler called")
	switch r.Method {
	case http.MethodGet:

		w.Header().Add("content-type", "application/json")
		output := parseFile("./internal/assets/norwegianScience.json")

		_, err := fmt.Fprint(w, string(output))
		if err != nil {
			log.Fatal("error in StubUniHandler")
		}
		break
	default:
		http.Error(w, "Method not supported", http.StatusNotImplemented)
	}
}

func STUBServer(group *sync.WaitGroup, port string) {
	defer group.Done()

	handlers := map[string]func(http.ResponseWriter, *http.Request){
		consts.StubDomain: StubHandler,
	}

	for path, function := range handlers {
		http.HandleFunc(path, function)
	}
	log.Println("STUB service running on port", port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
