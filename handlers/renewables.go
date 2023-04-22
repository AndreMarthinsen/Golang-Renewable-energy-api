package handlers

import (
	"Assignment2/caching"
	"Assignment2/consts"
	"Assignment2/util"
	"net/http"
	"strconv"
	"strings"
	//"time"
)

// Internal - paths
const currentPath = "current"
const historyPath = "history"

// RenewableStatistics struct that encapsulates information that will be returned for a successful request
type RenewableStatistics struct {
	Name       string  `json:"name"`
	Isocode    string  `json:"isocode"`
	Year       int     `json:"year,omitempty"` // if empty, will not be encoded in the response
	Percentage float64 `json:"percentage"`
}

// HandlerRenew Handler for the renewables endpoint: this checks if the request is GET, and calls the correct function
// for current renewable percentage or historical renewable percentage
func HandlerRenew(cfg *util.Config, request chan caching.CacheRequest, dataset map[string]util.Country, invocation chan []string, sortedYears map[string][]int) func(http.ResponseWriter, *http.Request) {
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
				handlerCurrent(cfg, w, r, strings.ToUpper(path[1]), request, dataset, invocation, sortedYears)
			case historyPath:
				handlerHistorical(cfg, w, r, strings.ToUpper(path[1]), dataset, invocation, sortedYears)
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
func handlerCurrent(cfg *util.Config, w http.ResponseWriter, r *http.Request, code string, request chan caching.CacheRequest, dataset map[string]util.Country, invocation chan []string, sortedYears map[string][]int) {
	var stats []RenewableStatistics
	// If the empty string is passed, all countries will be returned
	// Otherwise, tries to find country matching code in dataset
	cfg.DatasetLock.Lock()
	if code == "" {
		for key, val := range dataset {
			stats = append(stats, RenewableStatistics{
				val.Name,
				key,
				// Sets the year, i.e. the current year, to be the last year for which we have data in the dataset
				sortedYears[key][len(sortedYears[key])-1],
				val.YearlyPercentages[sortedYears[key][len(sortedYears[key])-1]],
			})
		}
	} else {
		if _, ok := dataset[code]; !ok {
			http.Error(w, "Code mispelled or country not in dataset", http.StatusNotFound)
			return
		}
		//TODO: invocation is put here for testing. Unsure of proper placement.
		invocation <- []string{code}
		stats =
			append(stats, RenewableStatistics{
				dataset[code].Name,
				code,
				// Sets the year, i.e. the current year, to be the last year for which we have data in the dataset
				sortedYears[code][len(sortedYears[code])-1],
				dataset[code].YearlyPercentages[sortedYears[code][len(sortedYears[code])-1]]},
			)

	}
	cfg.DatasetLock.Unlock()
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
			if result.Status != 404 {
				//TODO: invocation is put here for testing. Unsure of proper placement.
				invocation <- result.Neighbours[code]
				for _, val := range result.Neighbours[code] {
					stats = append(stats, RenewableStatistics{
						dataset[val].Name,
						val,
						sortedYears[val][len(sortedYears[val])-1],
						dataset[val].YearlyPercentages[sortedYears[val][len(sortedYears[val])-1]]},
					)
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
func handlerHistorical(cfg *util.Config, w http.ResponseWriter, r *http.Request, code string, dataset map[string]util.Country, invocation chan []string, sortedYears map[string][]int) {
	var stats []RenewableStatistics
	// if no code is provided, a list of every country's average renewable percentage is returned
	if code == "" {
		for key, val := range dataset {
			// sets up a statistic for each country in the dataset
			statistic := RenewableStatistics{Name: val.Name, Isocode: key, Year: 0}
			percentage := 0.0
			yearSpan := 0.0
			//calculates the average renewable percentage by iterating over that country's map of year to percentage pairs
			for _, year := range sortedYears[key] {
				percentage += dataset[key].YearlyPercentages[year]
				yearSpan++
			}
			percentage /= yearSpan
			statistic.Percentage = percentage
			stats = append(stats, statistic)
		}
	} else {
		if _, ok := dataset[code]; !ok {
			http.Error(w, "Code mispelled or country not in dataset", http.StatusNotFound)
			return
		}
		//TODO: invocation is put here for testing. Unsure of proper placement.
		invocation <- []string{code}
		// set start and end to match first and last year in dataset
		start := sortedYears[code][0]
		end := sortedYears[code][len(sortedYears[code])-1]
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
			if start < sortedYears[code][0] {
				start = sortedYears[code][0]
			}
			// If end has been set as higher than the last year in the dataset,
			// it is instead set to the last year
			// TODO: consider setting this as as bad request error instead
			if end > sortedYears[code][len(sortedYears[code])-1] {
				end = sortedYears[code][len(sortedYears[code])-1]
			}
		}
		country := RenewableStatistics{Name: dataset[code].Name, Isocode: code}
		// Adds yearly percentages for span from start to end
		// if not set by user, it will be from the first to the last year in the dataset
		for i := start; i <= end && i <= sortedYears[code][len(sortedYears[code])-1]; i++ {
			country.Year = i
			country.Percentage = dataset[code].YearlyPercentages[i]
			stats = append(stats, country)
		}
	}
	if len(stats) == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}
