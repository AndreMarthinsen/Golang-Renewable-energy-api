package handlers

import (
	"Assignment2/consts"
	"Assignment2/fsutils"
	"Assignment2/util"
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNotificationHandler(t *testing.T) {
	config, _ := util.SetUpServiceConfig(consts.ConfigPath, "../cmd/sha.json")
	var countryDB util.CountryDataset
	log.Println(os.Getwd())
	err := countryDB.Initialize("../internal/assets/renewable-share-energy.csv")
	if err != nil {
		t.Error(err)
		return
	}

	handler := NotificationHandler(&config, &countryDB)
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()
	client := http.Client{}
	defer client.CloseIdleConnections()

	doRequest := func(method string, path string, reader io.Reader) (*http.Response, error) {
		req, err := http.NewRequest(method, server.URL+path, reader)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(req)
		return response, nil
	}

	// Sending malformed body
	reader := strings.NewReader("{\n    invalid body asd1  3: 5\n}")
	response, err := doRequest(http.MethodPost, consts.NotificationPath, reader)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t,
		util.StatusToString(http.StatusBadRequest),
		response.Status,
	)

	// Sending correct body
	testWebhook := Webhook{
		URL:     "https://tullogtoys.crumb",
		Country: "NOR",
		Calls:   5,
	}
	bytestream, err := json.Marshal(testWebhook)
	if err != nil {
		t.Error(err)
	}

	response, err = doRequest(http.MethodPost, consts.NotificationPath, bytes.NewReader(bytestream))
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t,
		strconv.Itoa(http.StatusOK)+" "+http.StatusText(http.StatusOK),
		response.Status,
	)

	webhookId := WebhookRegResp{}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&webhookId)
	if err != nil {
		t.Error(err)
	}
	pathWithID := consts.NotificationPath + "/" + webhookId.WebhookId
	response, err = doRequest(http.MethodGet, pathWithID, nil)
	if err != nil {
		t.Error(err)
	}

	webhookLookup := Webhook{}
	decoder = json.NewDecoder(response.Body)
	err = decoder.Decode(&webhookLookup)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, testWebhook, webhookLookup)

	// Deletes a first time, should result in 200
	response, err = doRequest(http.MethodDelete, pathWithID, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, util.StatusToString(http.StatusOK), response.Status)

	// Attempts to delete again, should result in 404
	response, err = doRequest(http.MethodDelete, pathWithID, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, util.StatusToString(http.StatusNotFound), response.Status)

	// Attempts to delete with non-valid path
	// Attempts to delete again, should result in 404
	response, err = doRequest(http.MethodDelete, pathWithID+"/testpath", nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, util.StatusToString(http.StatusBadRequest), response.Status)
	// Attempts to show the now deleted webhook
	response, err = doRequest(http.MethodGet, pathWithID, nil)
	if err != nil {
		t.Error(err)
	}

	// Webhook should be deleted, which should cause decoding to fail.
	webhookLookup = Webhook{}
	decoder = json.NewDecoder(response.Body)
	err = decoder.Decode(&webhookLookup)
	if err == nil {
		t.Error(err)
	}
	assert.Equal(t, util.StatusToString(http.StatusNotFound), response.Status)

	// Uses an invalid path, should result in 400 Bad Request
	response, err = doRequest(http.MethodGet, consts.NotificationPath+"/err/or", nil)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, util.StatusToString(http.StatusBadRequest), response.Status)

	// returns all webhooks, should result in 200 OK
	response, err = doRequest(http.MethodGet, consts.NotificationPath, nil)
	if err != nil {
		t.Error(err)
	}
	webhooks := make([]Webhook, 0)
	decoder = json.NewDecoder(response.Body)
	err = decoder.Decode(&webhooks)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, util.StatusToString(http.StatusOK), response.Status)
	// checks that all webhooks in the collection is returned in the response
	webhookCount, err := fsutils.CountDocuments(&config, config.WebhookCollection)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, webhookCount, len(webhooks))

}
