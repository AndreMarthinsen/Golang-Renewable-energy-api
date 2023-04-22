package notifications

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
	"time"
)

// HandlerNotification The handler for the notification endpoint
func HandlerNotification(cfg *util.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		defer client.CloseIdleConnections()

		ctx := &util.HandlerContext{Name: "Notification handler: ", Writer: &w, Client: client}

		switch r.Method {
		case http.MethodPost:
			registerWebhook(ctx, cfg, r)
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
// and provides a response:
//
//	{
//	    "webhook_id": "<doc_ID_here>"
//	}
func registerWebhook(handler *util.HandlerContext, cfg *util.Config, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	webhook := WebhookRegistration{}
	if err := decoder.Decode(&webhook); err != nil {
		errorMsg :=
			"Malformed request body. Expected json format is:\n\n" +
				"{\n" +
				"    \"url\": \"https://localhost:8080/client/\",\n" +
				"    \"country\": \"NOR\",\n" +
				"    \"calls\": 5 <-- should trigger every five calls\n" +
				"}\n"
		http.Error(*handler.Writer, errorMsg, http.StatusBadRequest)
		return
	}
	newWebhookID, err := fsutils.AddDocument(cfg, cfg.WebhookCollection, &webhook)
	if err != nil {
		http.Error(*handler.Writer, "Failed to register your webhook", http.StatusBadRequest)
		return
	}
	util.EncodeAndWriteResponse(handler.Writer, WebhookRegResp{newWebhookID})
}

// deleteWebhook takes a request on the form
// Method: DELETE
// Path: /energy/v1/notifications/{id},
// and deletes a webhook if it is correctly identified.
// TODO: Response is up to us. Should not expose any vital information.
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
		webhookEntry := Webhook{}
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
		//TODO: Should it return an array for both branches for consistency?
		util.EncodeAndWriteResponse(handler.Writer, webhookEntry)
		return
	} else {
		iter := cfg.FirestoreClient.Collection(cfg.WebhookCollection).Documents(*cfg.Ctx)
		entries := make([]Webhook, 0)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			webhookEntry := Webhook{}
			if err = doc.DataTo(&webhookEntry); err != nil {
				log.Printf("Failed to unmarshal document %v: %v", doc.Ref.ID, err)
				continue
			}
			entries = append(entries, webhookEntry)
		}
		if len(entries) == 0 {
			http.Error(*handler.Writer,
				"",
				http.StatusNotFound, // Error indicates a failure to communicate
			) // with DB. Document not existing returns no error.
		}
		util.EncodeAndWriteResponse(handler.Writer, entries)
	}
}

// webhookTrigger triggers whenever x amount of invocations on
// a registered webhook country as occurred.
//
//	{
//	   "webhook_id": "OIdksUDwveiwe",
//	   "country": "Norway",
//	   "calls": 10      <-- Should be some multiple of registered call frequency, i.e. 2*5 in this case.
//	}
func webhookTrigger(context *util.HandlerContext, cfg *util.Config, w http.ResponseWriter, r *http.Request) {

}
