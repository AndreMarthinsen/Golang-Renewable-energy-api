// Package util contains generic functionality for use with the server handlers, such as
// url processing, outbound request handling, and encoding of responses.

package util

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"

	// "os"
	"net"
	"net/http"
	"strings"
	"time"
)

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
	DatasetLock       sync.Mutex
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

// InitializeDataset reads a csv-file line by line and fills a map of Country structs
// with information,
func InitializeDataset(path string) (map[string]Country, error) {
	code := ""
	dataset := make(map[string]Country)
	//sortedYears = make(map[string][]int)
	nr := ReadCSV(path)
	for {
		record, err := nr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			return make(map[string]Country), err
		}
		if record[1] != code {
			code = record[1]
			dataset[record[1]] = Country{Name: record[0], YearlyPercentages: make(map[int]float64)}
		} else if record[1] == code && record[1] != "" {
			year, _ := strconv.Atoi(record[2])
			dataset[code].YearlyPercentages[year], _ = strconv.ParseFloat(record[3], 32)
		}
	}
	// removes all entries where the code is not three characters long or where the code is the empty string
	// could perhaps be eliminated with, but eliminates invalid entries
	// also sorts the yearly renewable percentages for each Country
	for key := range dataset {
		/*sortedYears[key] = make([]int, 0)
		for year := range val.YearlyPercentages {
			sortedYears[key] = append(sortedYears[key], year)
		}
		sort.Ints(sortedYears[key])*/
		if key == "" || len(key) > 3 {
			delete(dataset, key)
		}
	}
	return dataset, nil
}

// ReadCSV attempts to open a CSV file and return a CSV-reader for that file
// if unsuccessful, the program crashes
func ReadCSV(path string) *csv.Reader {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("File error: %v\n", err)
	}
	nr := csv.NewReader(f)

	return nr
}

func SortDataset(dataset map[string]Country) map[string][]int {
	sortedYears := make(map[string][]int)
	for key, val := range dataset {
		sortedYears[key] = make([]int, 0)
		for year := range val.YearlyPercentages {
			sortedYears[key] = append(sortedYears[key], year)
		}
		sort.Ints(sortedYears[key])
	}
	return sortedYears
}
