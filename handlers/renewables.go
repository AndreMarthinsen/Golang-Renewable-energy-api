package handlers

import (
	"Assignment2/caching"
	"Assignment2/consts"
	"Assignment2/util"
	"encoding/csv"
	//"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	//"time"
)

// RenewableStatistics struct that encapsulates information that will be returned for a successful request
type RenewableStatistics struct {
	Name       string `json:"name"`
	Isocode    string `json:"isocode"`
	Year       string `json:"year,omitempty"` // if empty, will not be encoded in the response
	Percentage string `json:"percentage"`
}

// Dataset values TODO: Write csv.parsing function that can initialize
const lastYearString = "2021"
const lastYear = 2021
const firstYear = 1965
const yearSpan = lastYear - firstYear

// Internal - paths
const currentPath = "current"
const historyPath = "history"
const dataSetPath = "./internal/assets/renewable-share-energy.csv"

// HandlerRenew Handler for the renewables endpoint: this checks if the request is GET, and calls the correct function
// for current renewable percentage or historical renewable percentage
func HandlerRenew(request chan caching.CacheRequest) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method { // switch for easy expansion
		case http.MethodGet:
			path := util.FragmentsFromPath(r.URL.Path, consts.RenewablesPath)

			// if path is empty after /renewables/, returns error message
			if len(path) == 0 {
				http.Error(w, "Not found, only /current/ and /history/ supported", http.StatusNotFound)
			} else if len(path) < 2 { // if no cc3a-code is supplied with request,
				path = append(path, "") // appends an empty string that tells handlers
				// to find infomation about all countries
			}
			// checks if path contains /current/ or /history/, if not error message
			switch path[0] {
			case currentPath:
				handlerCurrent(w, r, strings.ToUpper(path[1]), request)
			case historyPath:
				handlerHistorical(w, r, strings.ToUpper(path[1]))
			default:
				http.Error(w, "Not found, only /current/ and /history/ supported", http.StatusNotFound)
				return
			}
		default:
			http.Error(w, "Method not implemented, only GET requests are supported", http.StatusNotImplemented)
			return
		}
	}
}

// handlerCurrent handles requests for renewable energy percentage for the current year in one country,
// with possibility for returning the same information for that country's neighbours
func handlerCurrent(w http.ResponseWriter, r *http.Request, code string, request chan caching.CacheRequest) {
	var stats []RenewableStatistics
	// Tries to find countries matching code in dataset
	// if the empty string is passed, all countries will be returned
	stats =
		append(stats, readStatsFromFile(dataSetPath, lastYearString, code)...)
	// if no match is found for passed code, or if results are otherwise failed to be found
	// returns error
	if len(stats) == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
	}
	// If a neighbours query has been, attempts to parse into bool
	// TODO: Move query-parsing into own function
	if r.URL.RawQuery != "" {
		query := r.URL.Query()
		neighboursTrue, err := strconv.ParseBool(query.Get("neighbours"))
		if err != nil {
			http.Error(w, "Bad request, neighbours must equal true or false", http.StatusBadRequest)
		}
		if neighboursTrue {
			// sends a request to the cache worker
			ret := make(chan caching.CacheResponse)
			request <- caching.CacheRequest{ChannelRef: ret, CountryRequest: []string{code}}
			result := <-ret
			// if the request doesn't return not found, it will find those neighbours
			if result.Status != 404 {
				for _, val := range result.Neighbours[code] {
					stats = append(stats, readStatsFromFile(dataSetPath, lastYearString, val)...)
				}
			}
			// for _, val := range result.Neighbours {
			// 	stats = append(stats, readStatsFromFile(dataSetPath, lastYearString, val[])...)
			// }
			// var neighbours []country
			// context :=
			// 	util.HandlerContext{Name: "current", Writer: &w, Client: &http.Client{Timeout: 10 * time.Second}}
			// var URL string
			// // forms URL for request, formatted to either stubserver or external API
			// if consts.Development {
			// 	URL = consts.StubDomain + consts.CountryCodePath + countriesCode + strings.ToUpper(code)
			// } else {
			// 	URL = restCountries + consts.CountryCodePath + countriesCode + code + bordersAffix
			// }
			// // sends request to stubserver/API
			// msg, err := util.HandleOutgoing(
			// 	&context,
			// 	http.MethodGet,
			// 	URL,
			// 	nil,
			// 	&neighbours)
			// if err != nil {
			// 	log.Println(msg, err)
			// 	return
			// }
			// // if country has neighbours according to stub/API, tries to find them in dataset
			// if len(neighbours) > 0 {
			// 	for _, val := range neighbours[0].Neighbours {
			// 		stats = append(stats, readStatsFromFile(dataSetPath, lastYearString, val)...)
			// 	}
			// }

		}
	}
	// if no match is found for passed code, or if results are otherwise failed to be found
	// returns error
	if len(stats) == 0 {
		http.Error(w, "Not", http.StatusNotFound)
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

// handlerHistorical Handles requests for the history of renewable energy in one country,
// on a yearly basis. Has functionality for setting starting and ending year of renewables history
func handlerHistorical(w http.ResponseWriter, r *http.Request, code string) {
	var stats []RenewableStatistics
	// if no code is provided, a list of every country's average renewable percentage is returned
	if code == "" {
		stats = append(stats, readStatsFromFile(dataSetPath, lastYearString, code)...)
		for i, val := range stats {
			tmp := readPercentageFromFile(dataSetPath, val.Isocode)
			tmp = tmp / yearSpan
			stats[i].Percentage = strconv.FormatFloat(tmp, 'f', -1, 64)
			stats[i].Year = ""
		}
	} else {
		// set start and end to match first and last year in dataset
		start := firstYear
		end := lastYear
		// The following checks if there is a URL query, if its correctly formatted, and if
		// it is, it sets the bounds of the beginning and end of the country's energy history
		// TODO: put query-handling in its own function
		if r.URL.RawQuery != "" {
			var err error
			query := r.URL.Query()
			start, err = strconv.Atoi(query.Get("begin"))
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
			}
			end, err = strconv.Atoi(query.Get("end"))
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
			}
			// Sends error if end year has been set to higher than start year
			if start > end {
				http.Error(w, "Bad request, begin must be smaller than end", http.StatusBadRequest)
			}
			// If end has been set as higher than the last year in the dataset,
			// it is instead set to the last year
			// TODO: consider setting this as as bad request error instead
			if end > lastYear {
				end = lastYear
			}
		}
		// Adds yearly percentages for span from start to end
		// if not set by user, it will be from the first to the last year in the dataset
		for i := start; i <= end && i <= lastYear; i++ {
			stats = append(stats, readStatsFromFile(dataSetPath, strconv.Itoa(i), code)...)
		}
	}
	if len(stats) == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

// readStatsFromFile fetches information from a cvs.file specified by path,
// puts in a slice of renewableStats and returns that slice
func readStatsFromFile(p string, year string, code string) []RenewableStatistics {
	var statistics []RenewableStatistics
	nr := readCSV(p)
	for {
		record, err := nr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			return []RenewableStatistics{}
		}
		// if the year-field in the current line in the file matches the year argument
		// a new statistics struct, containing that line's information is appended to the slice
		if record[2] == year {
			// if a cc3a code has been passed, only lines with countries matching that code
			// will be encapsulated
			if code != "" {
				if record[1] == code {
					statistics = append(statistics, RenewableStatistics{record[0], record[1], record[2], record[3]})
				}
			} else {
				// if an empty string is passed as code, all lines with cc3a codes (i.e. only countries, not Africa)
				// will be encapsulated and appended
				if record[1] != "" {
					statistics = append(statistics, RenewableStatistics{record[0], record[1], record[2], record[3]})
				}
			}
		}
	}
	return statistics
}

// readPercentageFromFile parses a csv file (i.e., the dataset) and returns the sum
// of a given country's renewable energy percentages, found by cc3a matching
func readPercentageFromFile(p string, code string) float64 {
	var percentage float64
	nr := readCSV(p)
	for {
		record, err := nr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			return 0.0
		}
		// if the line in the file has a cc3a code matching the code-param,
		// its renewable energy percentage is added to the current sum
		if record[1] == code {
			per, _ := strconv.ParseFloat(record[3], 32)
			percentage += per
		}
	}
	return percentage
}

// readCSV attempts to open a CSV file and return a CSV-reader for that file
// if unsuccessful, the program crashes
func readCSV(path string) *csv.Reader {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("File error: %v\n", err)
	}
	nr := csv.NewReader(f)
	return nr
}
