package notifications

// Webhook provides the json structure for the expected request
// body of a webhook registration.
type Webhook struct {
	URL     string `json:"url"`
	Country string `json:"country"`
	Calls   int32  `json:"calls"`
}

// WebhookRegistration provides the document structure of a
// webhook registration. CallCount is the invocation
// count for the registered country last time the webhook registration
// was checked for a potential trigger.
//
// WARNING: CallCount MUST be updated in DB on an invocation check.
// if the current invocation count of NOR is 103 at the time of the previous check,
// invocations from previous update is current in-cache invocation count - previous count.
// If Calls is any multiple of the sum, send x messages to the registered url.
type WebhookRegistration struct {
	URL       string `json:"url"`
	Country   string `json:"country"`
	Calls     int32  `json:"calls"`
	CallCount int32  `firestore:"call_count"`
}

// WebhookRegResp provides the json structure of the response body
// upon registration of a valid webhook.
type WebhookRegResp struct {
	WebhookId string `json:"webhook_id"`
}
