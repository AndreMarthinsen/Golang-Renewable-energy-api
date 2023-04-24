// Package util contains generic functionality for use with the server handlers, such as
// url processing, outbound request handling, and encoding of responses.

package util

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"golang.org/x/exp/constraints"
	"io"
	"log"
	"strconv"

	// "os"
	"net"
	"net/http"
	"strings"
	"time"
)

// Max returns the largest value
func Max[K constraints.Ordered](val K, val2 K) K {
	if val > val2 {
		return val
	}
	return val2
}

// Min returns the smallest value
func Min[K constraints.Ordered](val K, val2 K) K {
	if val > val2 {
		return val2
	}
	return val
}

// Config contains project config.
type Config struct {
	CachePushRate     time.Duration // Cache is pushed to external DB with CachePushRate as its interval
	CacheTimeLimit    time.Duration // Cache entries older than CacheTimeLimit are purged upon loading
	WebhookEventRate  time.Duration // How often registered webhooks should be checked for event triggers
	DebugMode         bool          // toggles any extra debug features such as extra logging of events
	DevelopmentMode   bool          // Sets the service to use stubbing of external APIs
	Ctx               *context.Context
	FirestoreClient   *firestore.Client
	CachingCollection string
	PrimaryCache      string
	WebhookCollection string
}

// RenewableStatistics struct that encapsulates information that will be returned for a successful request
type RenewableStatistics struct {
	Name       string  `json:"name"`
	Isocode    string  `json:"isocode"`
	Year       int     `json:"year,omitempty"` // if empty, will not be encoded in the response
	Percentage float64 `json:"percentage"`
}

// HandlerContext is a container for the name, writer and client object associated with
// a handler body.
type HandlerContext struct {
	Name   string
	Writer *http.ResponseWriter
	Client *http.Client
}

// Country struct that encapsulates the information for one Country in the dataset
type Country struct {
	Name              string
	AveragePercentage float64
	StartYear         int
	EndYear           int
	YearlyPercentages map[int]float64
}

// FragmentsFromPath takes an incoming URL path and the path of a handler, removing the handler portion
// of the full path, returning the remaining path split on '/'. The function also replaces any whitespace
// with %20 for safety.
func FragmentsFromPath(Path string, rootPath string) []string {
	trimmedPath := strings.TrimPrefix(Path, rootPath)
	trimmedPath = strings.ReplaceAll(trimmedPath, " ", "%20")
	return strings.FieldsFunc(trimmedPath, func(c rune) bool { return c == '/' })
}

// GetDomainStatus sends a basic get request to the supplied URL and returns the response
// status, or status timeout/protocol error message when appropriate. Performs logging of
// error events.
// return: err, err/nil
func GetDomainStatus(URL string) (string, error) {
	response, err := http.Get(URL)
	var status string
	if err != nil {
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			log.Println(URL, " request timed out: ", err)
			status = "timed out contacting service"
		} else {
			log.Println(URL, " protocol error: ", err)
			status = "protocol error"
		}
	} else {
		status = response.Status
	}
	if response != nil { // response == nil in case of time out
		err = response.Body.Close()
		if err != nil {
			log.Println(URL, ": Failed to close body:", err)
		}
	}
	return status, err
}

// EncodeAndWriteResponse attempts to encode data as a json response. Data must be a pointer
// to an appropriate object suited for encoding as json. Logging and errors are handled
// within the function.
func EncodeAndWriteResponse(w *http.ResponseWriter, data interface{}) {
	encoder := json.NewEncoder(*w)
	if err := encoder.Encode(data); err != nil {
		log.Println("Encoding error:", err)
		http.Error(*w, "Error during encoding", http.StatusInternalServerError)
		return
	}
	http.Error(*w, "", http.StatusOK)
}

// HandleOutgoing takes a HandlerContext, request method, target url and a reader object along with a pointer
// to an object to be used for decoding.
// target point to an object expected to match the returned json body resulting from the request.
// returns:
// On failure: a string detailing in what step the error occurred for use with logging, along with error object
// On success: "", nil
func HandleOutgoing(handler *HandlerContext, method string, URL string, reader io.Reader, target interface{}) (string, error) {
	request, err := http.NewRequest(method, URL, reader)
	if err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "failed to create request", err
	}

	response, err := handler.Client.Do(request)
	if err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "request to" + request.URL.Path + " failed", err
	}

	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(target); err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "failed to decode", err
	}
	return "", nil
}

// LogOnDebug logs all argument items if Config.DebugMode == true
func LogOnDebug(cfg *Config, msg ...any) {
	if cfg.DebugMode {
		log.Println("dbg:", msg)
	}
}

// ParseHistoricQuery parses the URL query from a request to the historyRenewables-handler if any is present
// if an error is encountered is return, along with default values for int and bool
// otherwise correct values are returned as parsed from query and nil is returned for error
func ParseHistoricQuery(w http.ResponseWriter, r *http.Request, dataset *CountryDataset, code string) (int, int, bool, error) {
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
			begin, err = strconv.Atoi(query.Get("begin"))
			if err != nil {
				http.Error(w, "Bad request, begin must be a whole number", http.StatusBadRequest)
				return 0, 0, false, err
			} else if !ok && code != "" {
				begin = dataset.GetFirstYear(code)
			}
		}
		// tries to find end
		if _, ok := query["end"]; ok {
			end, err = strconv.Atoi(query.Get("end"))
			if err != nil {
				http.Error(w, "Bad request, begin must be a whole number", http.StatusBadRequest)
				return 0, 0, false, err
			}
		} else if !ok && code != "" {
			end = dataset.GetLastYear(code)
		}
		// Sends error if end year has been set to higher than begin year
		if begin > end {
			http.Error(w, "Bad request, begin must be smaller than end", http.StatusBadRequest)
			return 0, 0, sortByValue, err
		}
		// if begin is lower than the first year for which a country has data, begin is set to that year
		if code != "" && begin < dataset.GetFirstYear(code) {
			begin = dataset.GetFirstYear(code)
		}
		// If end has been set as higher than the last year in the dataset,
		// it is instead set to the last year
		if code != "" && end > dataset.GetLastYear(code) {
			end = dataset.GetLastYear(code)
		}
		return begin, end, sortByValue, nil
	} else if r.URL.RawQuery == "" && code != "" {
		// if there isn't a URL query and the code is for a specific country rather than empty string
		// then that country's first and last years in the dataset is returned
		return dataset.GetFirstYear(code), dataset.GetLastYear(code), false, nil
	} else {
		// if the code is the empty string and no query is present, default values is returned
		return 0, 0, false, nil
	}
}
