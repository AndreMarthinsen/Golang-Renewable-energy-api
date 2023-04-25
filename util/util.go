// Package util contains generic functionality for use with the server handlers, such as
// url processing, outbound request handling, and encoding of responses.

package util

import (
	"context"
	"encoding/json"
	firebase "firebase.google.com/go"
	"golang.org/x/exp/constraints"
	"google.golang.org/api/option"
	"io"
	"log"
	"strconv"

	// "os"
	"net"
	"net/http"
	"strings"
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

// SetUpServiceConfig initializes the firestore context and client, then reads configuration
// settings from file. In the event of no config file being found, default settings will
// be used. Failure to find a config file will be logged, but does not trigger an error.
// Only failing to set up the firestore context and client will lead to a fail/error.
//
// On success: Config struct with valid firestore pointers, nil
// On failure: Config with nil pointers, error
func SetUpServiceConfig(configPath string, credentials string) (Config, error) {
	ctx := context.Background()
	opt := option.WithCredentialsFile(credentials)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return Config{}, err
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = config.Initialize(configPath)
	if err != nil { // Allowable error, running service with default config.
		log.Println(err)
	}
	config.FirestoreClient = client
	config.Ctx = &ctx
	return config, nil
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
	if err != nil {
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			return strconv.Itoa(http.StatusRequestTimeout) + " " +
				http.StatusText(http.StatusRequestTimeout), nil
		} else {
			return strconv.Itoa(http.StatusServiceUnavailable) + " " +
				http.StatusText(http.StatusServiceUnavailable), nil
		}
	}
	err = response.Body.Close()
	return response.Status, err
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
