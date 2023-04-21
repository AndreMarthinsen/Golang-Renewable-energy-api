package caching

import (
	"Assignment2/handlers/notifications"
	"Assignment2/util"
	"bytes"
	"cloud.google.com/go/firestore"
	"encoding/json"
	"google.golang.org/api/iterator"
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
			//TODO: Max 20 operations per bulk operation
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
					//TODO: Handle error
				}
				webhook := notifications.WebhookRegistration{}
				if err = doc.DataTo(&webhook); err != nil {
					//TODO: Handle error
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
					//TODO: Handle error
				}
				webhooksToCheck = append(webhooksToCheck, webhookCheck{ID: doc.Ref.ID, Registration: webhook})
			}
			// Iterates through relevant webhooks and
			for i, wh := range webhooksToCheck {
				oldCount := wh.Registration.CallCount
				newCount := invocationMap[wh.Registration.Country].Count + oldCount
				previousTriggers := oldCount / wh.Registration.Calls
				triggers := newCount/wh.Registration.Calls - previousTriggers
				if triggers != 0 {
					for j := 0; int32(j) < triggers; j++ {
						//TODO: Use Evens thing to get country name
						message := notifications.WebhookTrigger{
							WebhookId:  wh.ID,
							Country:    wh.Registration.Country,
							TotalCalls: previousTriggers + int32(j)*wh.Registration.Calls,
						}
						payload, err := json.Marshal(message)
						if err != nil {
							//TODO: Handle error
						}
						request, err := http.NewRequest(http.MethodPost, wh.Registration.URL, bytes.NewBuffer(payload))
						if err != nil {
							//TODO: Handle error
						}
						response, err := client.Do(request)
						util.LogOnDebug(cfg, "invocation worker: response from recipient on trigger"+
							" for url "+request.URL.Path+" : "+response.Status)
					}
				}
				webhooksToCheck[i].Registration.CallCount = newCount
			}

			//TODO: Send updated ones back to DB
			bulkOperation.End()
			invocationMap = map[string]InvocationCounter{} // reset of counters
			//TODO: Should we bother updating the ones that don't trigger?

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
