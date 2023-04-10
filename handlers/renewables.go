package handlers

import (
	"Assignment2/consts"
	//"Assignment2/internal/stubbing"
	"Assignment2/util"
	"encoding/csv"
	"strings"

	//"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	//"strings"
)

type renewableStats struct {
	Name	   string //`json:"name"`
	Isocode    string //`json:"isocode"`
	Year	   string //`json:"year"`
	Percentage string //`json:"percentage"`
}

const Current = "current"
const currentYear = "2021"
const History = "history"
const dataSetPath = "internal/assets/renewable-share-energy.csv"
const neighboursPrefix = "neighbours="
const neighboursTrue = "TRUE"
const restCountries = "http://129.241.150.113:8080/v3.1/"
const countriesCode = "alpha/"
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
	//log.Println(path, len(path))
	
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
	//TODO: remove log.Println(stats)
	if len(stats) == 0 {
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
	if r.URL.RawQuery != "" {
		getNeighbours := strings.ToUpper(strings.TrimLeft(r.URL.RawQuery, neighboursPrefix))
		if getNeighbours == neighboursTrue {
			var borders map[string][]string
			if consts.Development {
				context := 
				util.HandlerContext{Name: "current", Writer: &w, Client: &http.Client{Timeout: 10 * time.Second}}
				URL := restCountries+countriesCode+s+bordField
				util.HandleOutgoing(&context, 
					http.MethodGet, 
					URL, 
					nil, 
					&borders)
				//log.Println("You are here", borders["borders"])
				for _, val := range borders["borders"] {
					stats = append(stats, readStatsFromFile(dataSetPath, currentYear, val)...)
				}
			} else {

			}
		} 
	}
	http.Header.Add(w.Header(), "content-type", "application/json")
	util.EncodeAndWriteResponse(&w, stats)
}

//handlerHistorical Handles requests for the history of renewable energy in one country,
//on a yearly basis. Has functionality for setting starting and ending year of renewables history
func handlerHistorical(w http.ResponseWriter, r *http.Request, s string) {
	//var stats []renewableStats
}

//readStatsFromFile fetches information from the renewable data set, 
// puts in a slice of renewableStats and returns that slice
func readStatsFromFile(p string, year string, code string) []renewableStats {
	var tmp []renewableStats
	f, err := os.Open(p)
	if err != nil {
		log.Fatalf("File error: %v\n", err)
	}
	nr := csv.NewReader(f)
	for {
        record, err := nr.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatal(err)
        }
		if record[2] == year {
			//TODO: remove fmt.Print(record)
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