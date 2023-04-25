// Package util contains generic functionality for use with the server handlers, such as
// url processing, outbound request handling, and encoding of responses.

package util

import (
	"context"
	"encoding/json"
	firebase "firebase.google.com/go"
	"fmt"
	"golang.org/x/exp/constraints"
	"google.golang.org/api/option"
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

// StatusToString returns a formatted string of the provided error code.
// Returns empty string if unknown error code.
func StatusToString(status int) string {
	statusText := http.StatusText(status)
	if statusText != "" {
		return fmt.Sprintf("%d %v", status, http.StatusText(status))
	}
	return ""
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

// LogOnDebug logs all argument items if Config.DebugMode == true
func LogOnDebug(cfg *Config, msg ...any) {
	if cfg.DebugMode {
		log.Println("dbg:", msg)
	}
}
