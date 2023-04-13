package handlers

import (
	"Assignment2/consts"
	"Assignment2/util"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// struct that encapsulates information that will be returned for a successful request
type renewableStatistics struct {
	Name	   string `json:"name"`
	Isocode    string `json:"isocode"`
	Year	   string `json:"year,omitempty"` // if empty, will not be encoded in the response
	Percentage string `json:"percentage"`
}

// contains a country's neighbouring countries, used in decoding response
// from restcountries/stubbing
type country struct {
	Neighbours    []string `json:"borders"`
}

// Dataset values TODO: Write csv.parsing function that can initialize
const lastYearString = "2021"
const lastYear = 2021
const firstYear = 1965
const yearSpan = lastYear - firstYear

// Internal - paths
const currentPath = "current"
const historyPath = "history"
const dataSetPath = "internal/assets/renewable-share-energy.csv"

// External - paths
const restCountries = "http://129.241.150.113:8080/"
//const stubCodeAffix = "?codes="
const countriesCode = "?codes="
const bordersAffix = "&fields=borders"

// HandlerRenew Handler for the renewables endpoint: this checks if the request is GET, and calls the correct funtion
// for current renewable percentage or historical renewable percentage
func HandlerRenew(w http.ResponseWriter, r *http.Request) {
	switch r.Method { // switch for easy expansion
	case http.MethodGet:
		path := util.FragmentsFromPath(r.URL.Path, consts.RenewablesPath)

		// if path is empty after /renewables/, returns error message
		if len(path) == 0 {
			http.Error(w, "Not found, only /current/ and /history/ supported", http.StatusNotFound)
		} else if len(path) < 2 { 	// if no cc3a-code is supplied with request,
			path = append(path, "") // appends an empty string that tells handlers
									// to find infomation about all countries
		}
		// checks if path contains /current/ or /history/, if not error message
		switch path[0] {
		case currentPath: handlerCurrent(w, r, path[1])
		case historyPath: handlerHistorical(w, r, path[1])
		default: http.Error(w, "Not found, only /current/ and /history/ supported", http.StatusNotFound)
		}
	default: http.Error(w, "Method not implemented, only GET requests are supported", http.StatusNotImplemented)
	}
}

// handlerCurrent handles requests for renewable energy percentage for the current year in one country,
// with possibility for returning the same information for that country's neighbours
func handlerCurrent(w http.ResponseWriter, r *http.Request, code string) {
	var stats []renewableStatistics
	// Tries to find countries matching code in dataset
	// if the emtpy string is passed, all countries will be returned
	stats = 
	append(stats, readStatsFromFile(dataSetPath, lastYearString, strings.ToUpper(code))...)
	//
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
			var neighbours []country
			context := 
			util.HandlerContext{Name: "current", Writer: &w, Client: &http.Client{Timeout: 10 * time.Second}}
			var URL string
			// forms URL for request, formatted to either stubserver or external API
			if consts.Development {
				URL = consts.StubDomain + consts.CountryCodePath + countriesCode + strings.ToUpper(code)
			} else {
				URL = restCountries+consts.CountryCodePath+countriesCode+code+bordersAffix
			}
			// sends request to stubserver/API
			msg, err := util.HandleOutgoing(
				&context, 
				http.MethodGet, 
				URL, 
				nil, 
				&neighbours)
			if err != nil {
				fmt.Fprintf(w, msg, err)
			}
			// if country has neighbours according to stub/API, tries to find them in dataset
			if len(neighbours) > 0 {
				for _, val := range neighbours[0].Neighbours {
					stats = append(stats, readStatsFromFile(dataSetPath, lastYearString, val)...)
				}
			}
			
		}
	}
	if len(stats) == 0 {
		http.Error(w, "Not", http.StatusNotFound)
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

//handlerHistorical Handles requests for the history of renewable energy in one country,
//on a yearly basis. Has functionality for setting starting and ending year of renewables history
func handlerHistorical(w http.ResponseWriter, r *http.Request, code string) {
	var stats []renewableStatistics
	// if no code is provided, a list of every country's average renewable percentage is returned
	if code == "" {
		stats = append(stats, readStatsFromFile(dataSetPath, lastYearString, strings.ToUpper(code))...)
		for i, val := range stats {
			tmp := readPercentageFromFile(dataSetPath, val.Isocode)
			tmp = tmp/yearSpan
			stats[i].Percentage = strconv.FormatFloat(tmp, 'f', -1, 64)
			stats[i].Year = ""
		}
	} else {
		// set start and end to match 
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
			stats = append(stats, readStatsFromFile(dataSetPath, strconv.Itoa(i), strings.ToUpper(code))...)
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
func readStatsFromFile(p string, year string, code string) []renewableStatistics {
	var statistics []renewableStatistics
	nr := readCSV(p)
	for {
        record, err := nr.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatal(err)
        }
		// if the year-field in the current line in the file matches the year argument
		// a new statistics struct, containing that line's information is appended to the slice
		if record[2] == year {
			// if a cc3a code has been passed, only lines with countries matching that code
			// will be encapsulated
			if code != "" {
				if record[1] == code {
					statistics = append(statistics, renewableStatistics{record[0], record[1], record[2], record[3]})
				}
			} else {
				// if an emtpy string is passed as code, all lines with cc3a codes (i.e. only countries, not Africa)
				// will be encapsulated and appended
				if record[1] != "" {
					statistics = append(statistics, renewableStatistics{record[0], record[1], record[2], record[3]})
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
			log.Fatal(err)
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

func readCSV(p string) *csv.Reader {
	f, err := os.Open(p)
	if err != nil {
		log.Fatalf("File error: %v\n", err)
	}
	nr := csv.NewReader(f)
	return nr
}