package handlers

import (
	"bytes"
	"log"
	"net/http"
	"os"
)

const manualPath = "./internal/assets/HandlersManual.html"
const redirectPath = "./internal/assets/redirect.html"

// InfoHandler returns a html manual detailing the use of the various endpoints.
// Intended for use on the root path.
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	http.Header.Add(w.Header(), "content-type", "text/html")
	html, err := os.ReadFile(manualPath)
	if err != nil {
		log.Println("info handler: failed to read html body")
		http.Error(w, "Something went wrong...", http.StatusInternalServerError)
		return
	}
	bytes.NewReader(html)
	_, err = w.Write(html)
	if err != nil {
		log.Println("info handler: failed to write response")
		http.Error(w, "Something went wrong...", http.StatusInternalServerError)
		return
	}
}

// InvalidPathHandler displays a short html body offering a user to visit the
// handler manual page.
func InvalidPathHandler(w http.ResponseWriter, r *http.Request) {
	http.Header.Add(w.Header(), "content-type", "text/html")
	html, err := os.ReadFile(redirectPath)
	if err != nil {
		log.Println("info handler: failed to read html body")
		http.Error(w, "Something went wrong...", http.StatusInternalServerError)
		return
	}
	bytes.NewReader(html)
	_, err = w.Write(html)
	if err != nil {
		log.Println("info handler: failed to write response")
		http.Error(w, "Something went wrong...", http.StatusInternalServerError)
		return
	}
}
