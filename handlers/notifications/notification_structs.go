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
// count for the country since the registration of the webhook.
//
// WARNING: CallCount MUST be updated in DB on an invocation check.
type WebhookRegistration struct {
	URL       string `firestore:"url"`
	Country   string `firestore:"country"`
	Calls     int32  `firestore:"calls"`
	CallCount int32  `firestore:"call_count"`
}

// WebhookTrigger contains the information to be sent to the url of a registered
// webhook upon it being triggered.
type WebhookTrigger struct {
	WebhookId  string `json:"webhook_id"`
	Country    string `json:"country"`
	TotalCalls string `json:"calls"`
}

// WebhookRegResp provides the json structure of the response body
// upon registration of a valid webhook.
type WebhookRegResp struct {
	WebhookId string `json:"webhook_id"`
}
