package handlers

import (
	"Assignment2/consts"
	"Assignment2/util"
	"log"
	"net/http"
)

const Current = "current"
const History = "history"

// HandlerRenew Handler for the renewables endpoint: this checks if the request is GET, and calls the correct funtion
// for current renewable percentage or historical renewable percentage
func HandlerRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed, only GET requests are supported", http.StatusNotImplemented)
	}
	path := util.FragmentsFromPath(r.URL.Path, consts.RenewablesPath)
	query := ""
	log.Println(path)
	if r.URL.RawQuery != nil {
		query = r.URL.RawQuery
	}
	//TODO Implement handler for historical renewable percentages
	switch path[0] {
	case Current: handlerCurrent(w, r, path)
	case History: http.Error(w, "Bad request", http.StatusBadRequest)
	default: http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

//handlerCurrent Handles requests for renewable energy percentage for the current year in one country,
//with possibility for returning the same information for that country's neighbours
func handlerCurrent(w http.ResponseWriter, r *http.Request, s []string) {

}