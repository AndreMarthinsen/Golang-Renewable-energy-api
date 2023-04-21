package caching

import (
	"Assignment2/handlers/notifications"
	"Assignment2/util"
	"bytes"
	"cloud.google.com/go/firestore"
	"encoding/json"
	"google.golang.org/api/iterator"
	"log"
	"net/http"
	"time"
)

type InvocationCounter struct {
	Count   int32
	Updated bool
}

// InvocationWorker receives updates from endpoint handlers and updates
// an in memory data structure mapping country code to invocation count.
// Registered webhooks are periodically checked in DB to see if they should
// trigger, and if so, a message is sent to the registered url.
func InvocationWorker(cfg *util.Config, stop chan struct{}, invocationChannel chan []string) {

	invocationMap := make(map[string]InvocationCounter, 0)
	client := http.Client{}

	for {
		select {
		case <-time.After(time.Second * 1): // TODO: Config setting
			var updatedCountries []string
			for cca3, counter := range invocationMap {
				if counter.Updated {
					updatedCountries = append(updatedCountries, cca3)
					counter.Updated = false
					invocationMap[cca3] = counter
				}
			}

			query := cfg.FirestoreClient.Collection(cfg.WebhookCollection).Where("country", "in", updatedCountries)
			iter := query.Documents(*cfg.Ctx)

			type webhookCheck struct {
				ID           string
				Registration notifications.WebhookRegistration
			}

			bulkOperation := cfg.FirestoreClient.BulkWriter(*cfg.Ctx)

			var webhooksToCheck []webhookCheck
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Println("notification worker: ", err)
					break
				}
				webhook := notifications.WebhookRegistration{}
				if err = doc.DataTo(&webhook); err != nil {
					log.Println("notification worker: ", err)
					break
				}
				_, err = bulkOperation.Update(
					doc.Ref,
					[]firestore.Update{
						{
							Path:  "call_count",
							Value: webhook.CallCount + invocationMap[webhook.Country].Count,
						},
					})
				if err != nil {
					log.Println("notification worker: ", err)
					break
				}
				webhooksToCheck = append(webhooksToCheck, webhookCheck{ID: doc.Ref.ID, Registration: webhook})
			}
			// Iterates through relevant webhooks and
			for i, webhook := range webhooksToCheck {
				oldCount := webhook.Registration.CallCount
				newCount := invocationMap[webhook.Registration.Country].Count + oldCount
				previousTriggers := oldCount / webhook.Registration.Calls
				triggers := newCount/webhook.Registration.Calls - previousTriggers
				if triggers != 0 {
					for j := 0; int32(j) < triggers; j++ {
						//TODO: Use Evens thing to get country name
						message := notifications.WebhookTrigger{
							WebhookId:  webhook.ID,
							Country:    webhook.Registration.Country,
							TotalCalls: previousTriggers + int32(j)*webhook.Registration.Calls,
						}
						payload, err := json.Marshal(message)
						if err != nil {
							log.Println("invocation worker: ", err)
							break // TODO: Best response to this?
						}
						request, err := http.NewRequest(http.MethodPost, webhook.Registration.URL, bytes.NewBuffer(payload))
						if err != nil {
							log.Println("invocation worker: ", err)
							break // TODO: Best response to this?
						}
						response, err := client.Do(request)
						util.LogOnDebug(cfg, "invocation worker: response from recipient on trigger"+
							" for url "+request.URL.Path+" : "+response.Status)
					}
				}
				webhooksToCheck[i].Registration.CallCount = newCount
			}

			bulkOperation.End()                            // Executes write operations
			invocationMap = map[string]InvocationCounter{} // reset of counters

		case invocations, ok := <-invocationChannel:
			if ok != true {
				// TODO: Shut down due to channel connection loss
				return
			} // updates invocation count and sets updated to true
			for _, invocation := range invocations {
				if oldCounter, ok := invocationMap[invocation]; ok {
					invocationMap[invocation] = InvocationCounter{Count: oldCounter.Count + 1, Updated: true}
				} else {
					invocationMap[invocation] = InvocationCounter{Count: 1, Updated: true}
				}
			}
		}
	}
}
