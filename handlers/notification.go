package handlers

import (
	"Assignment2/fsutils"
	"Assignment2/util"
	"encoding/json"
	"net/http"
	"time"
)

// Webhook provides the json structure for the expected request
// body of a webhook registration.
type Webhook struct {
	URL     string `json:"url"`
	Country string `json:"country"`
	Calls   int32  `json:"calls"`
}

// WebhookRegResp provides the json structure of the response body
// upon registration of a valid webhook.
type WebhookRegResp struct {
	WebhookId string `json:"webhook_id"`
}

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
	webhook := Webhook{}
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
func deleteWebhook(context *util.HandlerContext, cfg *util.Config, r *http.Request) {

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
func viewWebhooks(context *util.HandlerContext, cfg *util.Config, r *http.Request) {

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
