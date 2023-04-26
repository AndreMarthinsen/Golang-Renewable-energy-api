package caching

import (
	"Assignment2/util"
	"bytes"
	"cloud.google.com/go/firestore"
	"encoding/json"
	"errors"
	"google.golang.org/api/iterator"
	"log"
	"net/http"
	"time"
)

// webhookCheck encapsulates the ID of a webhook in the DB along
// with the fields of the document.
type webhookCheck struct {
	ID   string
	Body webhookRegistration
}

// WebhookRegistration provides the document structure of a
// webhook registration. Count is the invocation
// count for the country since the registration of the webhook.
//
// WARNING: Count MUST be updated in DB on an invocation check.
type webhookRegistration struct {
	URL     string `firestore:"url"`
	Country string `firestore:"country"`
	Calls   int32  `firestore:"calls"`
	Count   int32  `firestore:"call_count"`
}

// WebhookTrigger contains the information to be sent to the url of a registered
// webhook upon it being triggered.
type webhookTrigger struct {
	WebhookId  string `json:"webhook_id"`
	Country    string `json:"country"`
	TotalCalls int32  `json:"calls"`
}

// InvocationWorker receives updates from endpoint handlers and updates
// an in memory data structure mapping country code to invocation count.
// Registered webhooks are periodically checked in DB to see if they should
// trigger, and if so, a message is sent to the registered url.
func InvocationWorker(cfg *util.Config, stop chan struct{}, done chan struct{}, countryDB *util.CountryDataset, invocationChannel chan []string) {

	// maps cca3 codes to the invocation count for a current cycle.
	invocationCounts := make(map[string]int32, 0)

	client := http.Client{}
	// Worker will stop to synchronize with the webhook DB every X seconds
	// set in the server config. When not synchronizing and doing triggers
	// the worker will count up any invocations of countries on the API endpoints.
	for {
		select {
		case <-time.After(cfg.WebhookEventRate):
			if len(invocationCounts) != 0 {
				handleInvocations(cfg, &client, countryDB, invocationCounts)
				invocationCounts = map[string]int32{} // reset of counters
			}
		case <-stop:
			if len(invocationCounts) != 0 {
				handleInvocations(cfg, &client, countryDB, invocationCounts)
			}
			done <- struct{}{}
			break
		case invocations, ok := <-invocationChannel:
			if ok != true {
				// TODO: Shut down due to channel connection loss
				return
			} // updates invocation count and sets updated to true
			for _, invocation := range invocations {
				if _, ok = invocationCounts[invocation]; ok {
					invocationCounts[invocation] += 1
				} else {
					invocationCounts[invocation] = 1
				}
			}
		}
	}
}

// handleInvocations
func handleInvocations(cfg *util.Config, client *http.Client, countryDB *util.CountryDataset, invocationCounts map[string]int32) {
	totalInvocations := int32(0)
	for _, val := range invocationCounts {
		totalInvocations += val
	}
	invocationCounts[""] = totalInvocations
	if cfg.DebugMode {
		log.Println("handling invocations for ", len(invocationCounts), " invocations")
		for code := range invocationCounts {
			log.Println(code)
		}
	}
	updatedCountries := getUpdatedCountries(invocationCounts)
	// firestore queries using 'in' supports up to 30 entries.
	maxInSize := 30
	// chunks = count of request batches that has to be performed to complete sync.
	chunks := ((len(updatedCountries) - 1) / maxInSize) + 1
	for i := 0; i < chunks; i++ {
		// queries only on countries that have seen an update in invocations
		ref := cfg.FirestoreClient.Collection(cfg.WebhookCollection)
		query := ref.Where("country", "in",
			updatedCountries[i*maxInSize:util.Min((i+1)*maxInSize, len(invocationCounts))],
		)
		iter := query.Documents(*cfg.Ctx)
		// update is done as atomic bulk operations
		bulkOperation := cfg.FirestoreClient.BulkWriter(*cfg.Ctx)

		err, webhooksToCheck := updateCallCountsAndGetEvents(iter, bulkOperation, invocationCounts)
		if err != nil {
			log.Println("invocation worker:", err)
		}
		// Outbound messages done for all triggered webhooks
		for _, webhook := range webhooksToCheck {
			if err := doWebhookEvents(cfg, client, webhook, countryDB, invocationCounts); err != nil {
				log.Println("invocation worker: ", err)
			}
		}
		bulkOperation.End() // Executes write operations
	}
}

// getUpdatedCountries returns a list of all countries found in the map for use with
// firestore queries.
func getUpdatedCountries(invocations map[string]int32) []string {
	updatedCountries := make([]string, 0)
	for cca3 := range invocations {
		updatedCountries = append(updatedCountries, cca3)
	}
	return updatedCountries
}

// updateCallCountsAndGetEvents iterates through the documents for a batch of countries
// and updates the call_count of all documents.
// On success: nil, list of webhooks that have been triggered
// On failure: error, nil slice or partially constructed slice.
func updateCallCountsAndGetEvents(iter *firestore.DocumentIterator, bulkOperation *firestore.BulkWriter,
	invocationMap map[string]int32) (error, []webhookCheck) {

	var webhooksToCheck []webhookCheck
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err, webhooksToCheck
		}
		webhook := webhookRegistration{}
		if err = doc.DataTo(&webhook); err != nil {
			return err, webhooksToCheck
		}
		_, err = bulkOperation.Update(
			doc.Ref,
			[]firestore.Update{
				{Path: "call_count", Value: webhook.Count + invocationMap[webhook.Country]},
			})
		if err != nil {
			return err, webhooksToCheck
		}
		webhooksToCheck = append(webhooksToCheck, webhookCheck{ID: doc.Ref.ID, Body: webhook})
	}
	return nil, webhooksToCheck
}

// doWebhookEvents performs outgoing messaging for triggered webhooks.
// A separate message will be sent out for each multiple of the clients
// 'calls' value since the last check was done, where 'calls' how many
// calls should go to a specified endpoint before an event triggers.
// On success: nil
// On failure: error
func doWebhookEvents(cfg *util.Config, client *http.Client, webhook webhookCheck,
	countryDB *util.CountryDataset, invocations map[string]int32) error {

	oldCount := webhook.Body.Count
	newCount := invocations[webhook.Body.Country] + oldCount
	previousTriggers := oldCount / webhook.Body.Calls
	triggers := newCount/webhook.Body.Calls - previousTriggers
	if triggers != 0 {
		for j := 0; int32(j) < triggers; j++ {
			countryName, err := countryDB.GetFullName(webhook.Body.Country)
			if err != nil {
				log.Println("webhook worker: ", err)
			}
			message := webhookTrigger{
				WebhookId:  webhook.ID,
				Country:    countryName,
				TotalCalls: previousTriggers*webhook.Body.Calls + int32(j+1)*webhook.Body.Calls,
			}
			payload, err := json.Marshal(message)
			if err != nil {
				return err
			}
			request, err := http.NewRequest(http.MethodPost, webhook.Body.URL, bytes.NewBuffer(payload))
			if err != nil {
				return err
			}
			_, err = client.Do(request)
			if err != nil {
				return errors.New(err.Error())
			}
		}
	}
	return nil
}
