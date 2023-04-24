package handlers

import (
	"Assignment2/caching"
	"Assignment2/consts"
	"Assignment2/util"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	//"time"
)

// Internal - paths
const currentPath = "current"
const historyPath = "history"

// HandlerRenew Handler for the renewables endpoint: this checks if the request is GET, and calls the correct function
// for current renewable percentage or historical renewable percentage
func HandlerRenew(cfg *util.Config, request chan caching.CacheRequest, dataset *util.CountryDataset, invocation chan []string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method { // switch for easy expansion
		case http.MethodGet:
			path := util.FragmentsFromPath(r.URL.Path, consts.RenewablesPath)

			// if path is empty after /renewables/, returns error message
			if len(path) == 0 {
				http.Error(w, "Not found, only /current/ and /history/ supported", http.StatusNotFound)
			} else if len(path) < 2 { // if no cc3a-code is supplied with request,
				path = append(path, "") // appends an empty string that tells handlers
				// to find information about all countries
			}
			// checks if path contains /current/ or /history/, if not error message
			switch path[0] {
			case currentPath:
				handlerCurrent(cfg, w, r, strings.ToUpper(path[1]), request, dataset, invocation)
			case historyPath:
				handlerHistorical(cfg, w, r, strings.ToUpper(path[1]), dataset, invocation)
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
func handlerCurrent(cfg *util.Config, w http.ResponseWriter, r *http.Request, code string, request chan caching.CacheRequest, dataset *util.CountryDataset, invocation chan []string) {
	var stats []util.RenewableStatistics
	// If the empty string is passed, all countries will be returned
	// Otherwise, tries to find country matching code in dataset

	if code == "" {
		stats = dataset.GetStatistics()
	} else {
		// if code is longer than 3 characters it is treated as a name
		// if that name can be found in the dataset, the code variable is set to that country's cc3a code
		if len(code) > 3 {
			code = strings.ReplaceAll(code, "%20", " ")
			var err error
			code, err = dataset.GetCountryByName(code)
			if err != nil {
				http.Error(w, "404 not found", http.StatusNotFound)
				return
			}
		}
		statistic, err := dataset.GetStatistic(code)
		if err != nil {
			http.Error(w, "Code misspelled or country not in dataset", http.StatusNotFound)
			return
		}
		//TODO: invocation is put here for testing. Unsure of proper placement.
		invocation <- []string{code}

		stats = append(stats, statistic)
	}
	// if no match is found for passed code, or if results are otherwise failed to be found
	// returns error
	if len(stats) == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
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
			if result.Status != http.StatusNotFound {
				//TODO: invocation is put here for testing. Unsure of proper placement.
				invocation <- result.Neighbours[code]
				for _, neighbour := range result.Neighbours[code] {
					statistic, err := dataset.GetStatistic(neighbour)
					if err == nil {
						stats = append(stats, statistic)
					}
				}
			}
		}
	}
	// if no match is found for passed code, or if results have otherwise failed to be found
	// returns error
	if len(stats) == 0 {
		http.Error(w, "Not", http.StatusNotFound)
		return
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

// handlerHistorical Handles requests for the history of renewable energy in one country,
// on a yearly basis. Has functionality for setting starting and ending year of renewables history
func handlerHistorical(cfg *util.Config, w http.ResponseWriter, r *http.Request, code string, dataset *util.CountryDataset, invocation chan []string) {
	var stats []util.RenewableStatistics
	var begin, end int
	var sortByValue bool
	var err error
	// if no code is provided, a list of every country's average renewable percentage is returned
	if code == "" {
		begin, end, sortByValue, err = parseHistoricQuery(w, r, dataset, code)
		if err != nil {
			return
		}
		stats = dataset.GetHistoricStatistics()
		// if both begin and end queries have been specified, the averages for all countries are
		// calculated only for that span
		if begin != 0 && end != 0 {
			for i := range stats {
				percentage := 0.0
				percentage, err = dataset.CalculatePercentage(stats[i].Isocode, begin, end)
				if err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				stats[i].Percentage = percentage
			}
		}
		if sortByValue {
			sort.Slice(stats, func(i, j int) bool {
				return stats[i].Percentage < stats[j].Percentage
			})
		}
	} else {
		if len(code) > 3 {
			// if code is longer than three characters, then it is treated as a country name
			// tries to find country with that name
			code = strings.ReplaceAll(code, "%20", " ")
			code, err = dataset.GetCountryByName(code)
			if err != nil {
				http.Error(w, "404 not found", http.StatusNotFound)
				return
			}
		}
		if !dataset.HasCountryInRecords(code) {
			http.Error(w, "Code mispelled or country not in dataset", http.StatusNotFound)
			return
		}
		// parses
		begin, end, sortByValue, err = parseHistoricQuery(w, r, dataset, code)
		if err != nil {
			return
		}
		//TODO: invocation is put here for testing. Unsure of proper placement.
		invocation <- []string{code}
		log.Println(end)
		log.Println(begin)
		// Adds yearly percentages for span from begin to end
		// if not set by user, it will be from the first to the last year in the dataset
		stats = dataset.GetStatisticsRange(code, begin, end)
	}
	if len(stats) == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

// parseHistoricQuery parses the URL query from a request to the historyRenewables-handler if any is present
// if an error is encountered is return, along with default values for int and bool
// otherwise correct values are returned as parsed from query and nil is returned for error
func parseHistoricQuery(w http.ResponseWriter, r *http.Request, dataset *util.CountryDataset, code string) (int, int, bool, error) {
	var err error
	var begin int
	var end int
	var sortByValue bool
	// checks if there is a URL query
	if r.URL.RawQuery != "" {
		query := r.URL.Query()
		if _, ok := query["sortByValue"]; ok {
			sortByValue, err = strconv.ParseBool(query.Get("sortByValue"))
			if err != nil {
				http.Error(w, "Bad request, sortByValue must equal true or false", http.StatusBadRequest)
				return 0, 0, false, err
			}
		}
		if _, ok := query["begin"]; ok {
			// tries to find begin
			begin, err = strconv.Atoi(query.Get("begin"))
			if err != nil {
				http.Error(w, "Bad request, begin must be a whole number", http.StatusBadRequest)
				return 0, 0, false, err
			}
			// checks if begin has been set lower than a country's first year in dataset
			if code != "" {
				begin = util.Max(begin, dataset.GetFirstYear(code))
			}
			// if no query is found for begin, and the request was for specific country
			// then end is set to that country's first year in dataset
		} else if !ok && code != "" {
			begin = dataset.GetFirstYear(code)
		}
		// tries to find end
		if _, ok := query["end"]; ok {
			end, err = strconv.Atoi(query.Get("end"))
			if err != nil {
				http.Error(w, "Bad request, begin must be a whole number", http.StatusBadRequest)
				return 0, 0, false, err
			}
			// checks if end has been higher than a country's last year in dataset
			if code != "" {
				end = util.Min(end, dataset.GetLastYear(code))
			}
			// if no query is found for end, and the request was for specific country
			// then end is set to that country's last year in dataset
		} else if !ok && code != "" {
			end = dataset.GetLastYear(code)
		}
		// Sends error if end year has been set to higher than begin year
		if begin > end {
			if code == "" {
				http.Error(w, "Bad request, begin end must be specified for all countries", http.StatusBadRequest)
				return 0, 0, sortByValue, errors.New("bad request, begin and end must be specified for all countries")
			}
			http.Error(w, "Bad request, begin must be smaller than end", http.StatusBadRequest)
			return 0, 0, sortByValue, errors.New("bad request, begin must be smaller than end")
		}
		return begin, end, sortByValue, nil
	} else {
		// if the code is the empty string and no query is present, default values is returned
		if code == "" {
			return 0, 0, false, nil
		} else { // if code is not empty, first and last year for that country is returned
			return dataset.GetFirstYear(code), dataset.GetLastYear(code), false, nil
		}
	}
}
