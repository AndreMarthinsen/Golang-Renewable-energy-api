package handlers

import (
	"Assignment2/consts"
	"Assignment2/util"
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type renewableStats struct {
	Name	   string `json:"name"`
	Isocode    string `json:"isocode"`
	Year	   string `json:"year,omitempty"`
	Percentage string `json:"percentage"`
}

type country struct {
	Borders    []string `json:"borders"`
}

const Current = "current"
const currentYear = "2021"
const firstYear = "1965"
const yearSpan = 56
const History = "history"
const dataSetPath = "internal/assets/renewable-share-energy.csv"
const neighboursPrefix = "neighbours="
const neighboursTrue = "TRUE"
const restCountries = "http://129.241.150.113:8080/v3.1/"
const countriesCode = "alpha/"
const stubCodeAffix = "?codes="
const bordField = "?fields=borders"

// HandlerRenew Handler for the renewables endpoint: this checks if the request is GET, and calls the correct funtion
// for current renewable percentage or historical renewable percentage
func HandlerRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed, only GET requests are supported", http.StatusNotImplemented)
	}
	path := util.FragmentsFromPath(r.URL.Path, consts.RenewablesPath)
	if len(path) < 2 {
		path = append(path, "")
	}
		
	//TODO Implement handler for historical renewable percentages
	switch path[0] {
	case Current: handlerCurrent(w, r, path[1])
	case History: handlerHistorical(w, r, path[1])
	default: http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

//handlerCurrent Handles requests for renewable energy percentage for the current year in one country,
//with possibility for returning the same information for that country's neighbours
func handlerCurrent(w http.ResponseWriter, r *http.Request, s string) {
	var stats []renewableStats
    if s == "" {
		stats = 
		append(stats, readStatsFromFile(dataSetPath, currentYear, "")...)
	} else {
		stats = 
		append(stats, readStatsFromFile(dataSetPath, currentYear, strings.ToUpper(s))...)
	}
	// if len(stats) == 0 {
	// 	http.Error(w, "Bad request", http.StatusBadRequest)
	// }
	if r.URL.RawQuery != "" {
		getNeighbours := strings.ToUpper(strings.TrimLeft(r.URL.RawQuery, neighboursPrefix))
		if getNeighbours == neighboursTrue {
			var c []country
			context := 
			util.HandlerContext{Name: "current", Writer: &w, Client: &http.Client{Timeout: 10 * time.Second}}
			var URL string
			// TODO: refactor this into generic function that can handle both
			// single country and country slice
			if consts.Development {
				URL = consts.StubDomain + consts.CountryCodePath + stubCodeAffix + strings.ToUpper(s)
			} else {
				URL = restCountries+countriesCode+s//+bordField
			}
			util.HandleOutgoing(
				&context, 
				http.MethodGet, 
				URL, 
				nil, 
				&c)
			for _, val := range c[0].Borders {
				stats = append(stats, readStatsFromFile(dataSetPath, currentYear, val)...)
			}
		}
	}
	if len(stats) == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

//handlerHistorical Handles requests for the history of renewable energy in one country,
//on a yearly basis. Has functionality for setting starting and ending year of renewables history
func handlerHistorical(w http.ResponseWriter, r *http.Request, s string) {
	var stats []renewableStats
	if s == "" {
		stats = append(stats, readStatsFromFile(dataSetPath, currentYear, strings.ToUpper(s))...)
		for i, val := range stats {
			var tmp float64
			tmp = readPercentageFromFile(dataSetPath, val.Isocode)
			tmp = tmp/yearSpan
			stats[i].Percentage = strconv.FormatFloat(tmp, 'f', -1, 64)
			stats[i].Year = ""
		}
	} else {
		start,_  := strconv.Atoi(firstYear)
		end,_ := strconv.Atoi(currentYear)
		for i := start; i <= end; i++ {
			stats = append(stats, readStatsFromFile(dataSetPath, strconv.Itoa(i), strings.ToUpper(s))...)
		}
	}
	if len(stats) == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

//readStatsFromFile fetches information from a cvs.file specified by path, 
// puts in a slice of renewableStats and returns that slice
func readStatsFromFile(p string, year string, code string) []renewableStats {
	var tmp []renewableStats
	nr := readCSV(p)
	for {
        record, err := nr.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatal(err)
        }
		if record[2] == year {
			if code != "" {
				if record[1] == code {
					tmp = append(tmp, renewableStats{record[0], record[1], record[2], record[3]})
				}
			} else {
				if record[1] != "" {
					tmp = append(tmp, renewableStats{record[0], record[1], record[2], record[3]})
				}
			}			
		}
	}
	return tmp
}

func readPercentageFromFile(p string, /*year string,*/ code string) float64 {
var tmp float64
nr := readCSV(p)
for {
	record, err := nr.Read()
	if err == io.EOF {
		break
	}
	if err != nil {
		log.Fatal(err)
	}
	//if record[2] == year {
		if code != "" {
			if record[1] == code {
				per, _ := strconv.ParseFloat(record[3], 32) 
				tmp += per
			}
		} else {
			if record[1] != "" {
				per, _ := strconv.ParseFloat(record[3], 32)
				tmp += per
			}
		}			
	//}
}
return tmp
}

func readCSV(p string) *csv.Reader {
	f, err := os.Open(p)
	if err != nil {
		log.Fatalf("File error: %v\n", err)
	}
	nr := csv.NewReader(f)
	return nr
}