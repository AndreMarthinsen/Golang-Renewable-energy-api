package caching

import (
	"Assignment2/handlers/notifications"
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

type webhookCheck struct {
	ID   string
	Body notifications.WebhookRegistration
}

// InvocationWorker receives updates from endpoint handlers and updates
// an in memory data structure mapping country code to invocation count.
// Registered webhooks are periodically checked in DB to see if they should
// trigger, and if so, a message is sent to the registered url.
func InvocationWorker(cfg *util.Config, stop chan struct{}, countryDB *util.CountryDataset, invocationChannel chan []string) {

	// maps cca3 codes to the invocation count for a current cycle.
	invocationMap := make(map[string]int32, 0)

	client := http.Client{}

	for {
		select {
		case <-time.After(time.Second * 1): // TODO: Config setting
			if len(invocationMap) != 0 {
				log.Println("handling invocations for")
				for code := range invocationMap {
					log.Println(code)
				}
				updatedCountries := getUpdatedCountries(invocationMap)

				ref := cfg.FirestoreClient.Collection(cfg.WebhookCollection)
				query := ref.Where("country", "in", updatedCountries)
				iter := query.Documents(*cfg.Ctx)

				bulkOperation := cfg.FirestoreClient.BulkWriter(*cfg.Ctx)

				err, webhooksToCheck := getDocumentsToUpdate(iter, bulkOperation, invocationMap)
				if err != nil {
					log.Println("invocation worker:", err)
				}
				// Iterates through relevant webhooks and
				for i, webhook := range webhooksToCheck {
					if err := doWebhookEvents(cfg, &client, webhook, countryDB, invocationMap); err != nil {
						log.Println("invocation worker: ", err)
					} else {
						webhooksToCheck[i].Body.Count = invocationMap[webhook.Body.Country] + webhook.Body.Count
					}
				}

				bulkOperation.End()                // Executes write operations
				invocationMap = map[string]int32{} // reset of counters
			}
		case invocations, ok := <-invocationChannel:
			if ok != true {
				// TODO: Shut down due to channel connection loss
				return
			} // updates invocation count and sets updated to true
			for _, invocation := range invocations {
				if _, ok = invocationMap[invocation]; ok {
					invocationMap[invocation] += 1
				} else {
					invocationMap[invocation] = 1
				}
			}
		}
	}
}

// getUpdatedCountries retrieves a list of all countries that have been updated
func getUpdatedCountries(invocations map[string]int32) []string {
	updatedCountries := make([]string, 0)
	for cca3 := range invocations {
		updatedCountries = append(updatedCountries, cca3)
	}
	return updatedCountries
}

func getDocumentsToUpdate(iter *firestore.DocumentIterator, bulkOperation *firestore.BulkWriter,
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
		webhook := notifications.WebhookRegistration{}
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
			message := notifications.WebhookTrigger{
				WebhookId:  webhook.ID,
				Country:    countryName,
				TotalCalls: previousTriggers + int32(j)*webhook.Body.Calls,
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
