// Package util contains generic functionality for use with the server handlers, such as
// url processing, outbound request handling, and encoding of responses.

package util

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"io"
	"log"
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
	DebugMode         bool          // toggles any extra debug features such as extra logging of events
	DevelopmentMode   bool          // Sets the service to use stubbing of external APIs
	Ctx               *context.Context
	FirestoreClient   *firestore.Client
	CachingCollection string
	PrimaryCache      string
	WebhookCollection string
}

// HandlerContext is a container for the name, writer and client object associated with
// a handler body.
type HandlerContext struct {
	Name   string
	Writer *http.ResponseWriter
	Client *http.Client
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
