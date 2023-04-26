package handlers

import (
	"Assignment2/consts"
	"Assignment2/fsutils"
	"Assignment2/util"
	"encoding/json"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net/http"
	"strings"
	"time"
)

// NotificationHandler The handler for the notification endpoint
func NotificationHandler(cfg *util.Config, countryDB *util.CountryDataset) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		defer client.CloseIdleConnections()

		ctx := &util.HandlerContext{Name: "Notification handler: ", Writer: &w, Client: client}

		switch r.Method {
		case http.MethodPost:
			registerWebhook(ctx, cfg, r, countryDB)
		case http.MethodGet:
			viewWebhooks(ctx, cfg, r)
		case http.MethodDelete:
			deleteWebhook(ctx, cfg, r)
		}
	}
}

// registerWebhook takes a request on the form
// Method: Post
// Path: /energy/v1/notifications/,
// Body:
//
//	{
//	   "url": "https://localhost:8080/client/",
//	   "country": "NOR",
//	   "calls": 5 <-- should trigger every five calls
//	}
//
// and provides a response upon a successful registration in the firebase DB:
//
//	{
//	    "webhook_id": "<doc_ID_here>"
//	}
func registerWebhook(handler *util.HandlerContext, cfg *util.Config, r *http.Request, countryDB *util.CountryDataset) {
	decoder := json.NewDecoder(r.Body)
	webhook := WebhookRegistration{}
	webhookIsValid := true
	st := http.StatusOK
	if err := decoder.Decode(&webhook); err != nil {
		webhookIsValid = false
		st = http.StatusBadRequest
	} else {
		webhook.Country = strings.ToUpper(webhook.Country)
		countryValid := countryDB.HasCountryInRecords(webhook.Country)
		if !countryValid {
			cca3, err := countryDB.GetCountryByName(webhook.Country)
			if err == nil {
				webhook.Country = cca3
				countryValid = true
			}
		}
		webhookIsValid =
			(countryValid || webhook.Country == "") &&
				validateURL(webhook.URL) && webhook.Calls != 0
		if !webhookIsValid {
			st = http.StatusUnprocessableEntity
		}
	}

	if webhookIsValid {
		newWebhookID, err := fsutils.AddDocument(cfg, cfg.WebhookCollection, &webhook)
		if err != nil {
			http.Error(*handler.Writer,
				"Webhook is valid, but registration failed due to an unexpected error.",
				http.StatusInternalServerError)
			return
		}
		util.EncodeAndWriteResponse(handler.Writer, WebhookRegResp{newWebhookID})
	} else {
		errorMsg :=
			"Malformed request body or non-valid values.\n Expected json format is:\n\n" +
				"{\n" +
				"    \"url\": \"https://localhost:8080/client/\",\n" +
				"    \"country\": \"NOR\",\n" +
				"    \"calls\": 5\n" +
				"}\n\n" +
				"Zero value for calls is not permitted. Must be 1 and above.\n" +
				"Country must either be a valid cca3 code, the full country name, or an empty string.\n" +
				"An empty country field will cause any country invocation to count up calls."
		http.Error(*handler.Writer, errorMsg, st)
		return
	}
}

// validateURL validates the url of an incoming webhook registration.
// currently it only checks that it's not an empty string.
func validateURL(url string) bool {
	return url != ""
}

// deleteWebhook takes a request on the form
// Method: DELETE
// Path: /energy/v1/notifications/{id},
// and deletes a webhook if it is correctly identified.
func deleteWebhook(handler *util.HandlerContext, cfg *util.Config, r *http.Request) {
	segments := util.FragmentsFromPath(r.URL.Path, consts.NotificationPath)
	if len(segments) != 1 {
		http.Error(*handler.Writer,
			"Not a valid path. For deletion, use /energy/v1/notifications/{id}",
			http.StatusBadRequest,
		)
		return
	}
	_, err := fsutils.ReadDocument(cfg, cfg.WebhookCollection, segments[0])
	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.Error(*handler.Writer,
				"No webhook deleted",
				http.StatusNotFound, // Document doesn't exist.
			)
		} else {
			http.Error(*handler.Writer,
				"Something went wrong...",
				http.StatusInternalServerError, // Firestore interaction failed.
			)
		}
		return
	}
	if err := fsutils.DeleteDocument(cfg, cfg.WebhookCollection, segments[0]); err != nil {
		http.Error(*handler.Writer,
			"Something went wrong...",
			http.StatusInternalServerError, // Error indicates a failure to communicate
		) // with DB. Document not existing returns no error.
		return
	}
	http.Error(*handler.Writer, "", http.StatusOK)
}

// viewWebhooks takes a request on the form
// Method: GET
// Path: /energy/v1/notifications/{id?}
// with a response
// [
//
//	{
//	   "webhook_id": "OIdksUDwveiwe",
//	   "url": "https://localhost:8080/client/",
//	   "country": "NOR",
//	   "calls": 5
//	},
//	...
//
// ]
// in the case of a provided ID, only a single result will be shown.
func viewWebhooks(handler *util.HandlerContext, cfg *util.Config, r *http.Request) {
	segments := util.FragmentsFromPath(r.URL.Path, consts.NotificationPath)
	if len(segments) == 1 {
		id := segments[0]
		webhookEntry := WebhookDisplay{}
		err := fsutils.ReadDocumentGeneral(cfg, cfg.WebhookCollection, id, &webhookEntry)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				http.Error(*handler.Writer,
					"",
					http.StatusNotFound, // Document doesn't exist.
				)
			} else {
				http.Error(*handler.Writer,
					"Something went wrong...",
					http.StatusInternalServerError, // Firestore interaction failed.
				)
			}
			return
		}
		webhookEntry.WebhookId = id
		util.EncodeAndWriteResponse(handler.Writer, webhookEntry)
		return
	} else if len(segments) == 0 {
		iter := cfg.FirestoreClient.Collection(cfg.WebhookCollection).Documents(*cfg.Ctx)
		entries := make([]WebhookDisplay, 0)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			webhookEntry := WebhookDisplay{}
			if err = doc.DataTo(&webhookEntry); err != nil {
				log.Printf("Failed to unmarshal document %v: %v", doc.Ref.ID, err)
				continue
			}
			webhookEntry.WebhookId = doc.Ref.ID
			entries = append(entries, webhookEntry)
		}
		if len(entries) == 0 {
			http.Error(*handler.Writer,
				"",
				http.StatusNotFound, // Error indicates a failure to communicate
			) // with DB. Document not existing returns no error.
		}
		util.EncodeAndWriteResponse(handler.Writer, entries)
	} else {
		http.Error(*handler.Writer,
			"Invalid path.",
			http.StatusBadRequest, // Error indicates a failure to communicate
		) // with DB. Document not existing returns no error.
	}
}
