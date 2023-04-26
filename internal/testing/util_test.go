package testing

import (
	"Assignment2/consts"
	"Assignment2/util"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigInitialize(t *testing.T) {
	var testConfig util.Config

	testConfig.InitializeWithDefaults()
	defaultConfig := util.Config{
		CachePushRate:     util.SettingsCachePushRate,
		CacheTimeLimit:    util.SettingsCacheTimeLimit,
		DebugMode:         util.SettingsDebugMode,
		DevelopmentMode:   util.SettingsDevelopmentMode,
		CachingCollection: util.SettingsCachingCollection,
		PrimaryCache:      util.SettingsPrimaryCache,
		WebhookCollection: util.SettingsWebhookCollection,
		WebhookEventRate:  util.SettingsWebhookEventRate,
	}
	assert.Equal(t, defaultConfig, testConfig)
	assert.Nil(t, testConfig.Initialize(consts.ConfigPath))
	assert.Error(t, testConfig.Initialize("/invalid_path"))
}

func TestStatusToText(t *testing.T) {
	assert.Equal(t, "200 OK", util.StatusToString(http.StatusOK))
	assert.Equal(t, "404 Not Found", util.StatusToString(http.StatusNotFound))
	assert.Equal(t, "", util.StatusToString(3000))
}

func TestSetUpServiceConfig(t *testing.T) {
	panicTest := func(cfg *util.Config) func() {
		return func() {
			// panic on nil pointer dereference
			err := cfg.FirestoreClient.Close()
			if err != nil {
				t.Error(err)
			}
		}
	}

	config, err := util.SetUpServiceConfig("/invalid/path", "./sha.json")
	assert.Nil(t, err)
	assert.NotPanics(t, panicTest(&config), config)

	config, err = util.SetUpServiceConfig(consts.ConfigPath, "./invalid_creds.json")
	assert.Error(t, err)
	assert.Panics(t, panicTest(&config), config)

	config, err = util.SetUpServiceConfig(consts.ConfigPath, "./sha.json")
	assert.Nil(t, err)
	assert.NotPanics(t, panicTest(&config))

}

func TestMax(t *testing.T) {
	assert.Equal(t, 5, util.Max(5, 0))
	assert.Equal(t, 0, util.Max(-5, 0))
	assert.Equal(t, 0.0, util.Max(0.0, 0.0))
	assert.Equal(t, 'b', util.Max('b', 'a'))
}

func TestMin(t *testing.T) {
	assert.Equal(t, 0, util.Min(5, 0))
	assert.Equal(t, -5, util.Min(-5, 0))
	assert.Equal(t, 0.0, util.Min(0.0, 0.0))
	assert.Equal(t, 'a', util.Min('b', 'a'))
}

// TestFragmentsFromPath test that verifies parsing of an url string into a set of segments.
func TestFragmentsFromPath(t *testing.T) {
	runFragmentsTest := func(path string, rootPath string, expected []string) func(t *testing.T) {
		return func(t *testing.T) {
			processed := util.FragmentsFromPath(path, rootPath)
			if len(processed) != len(expected) {
				t.Error("Expected \n", expected, "\nbut got: \n", processed)
			}
		}
	}
	// defines a slice of test structs before running the above function on each struct.
	tests := []struct {
		name     string
		path     string
		rootPath string
		expected []string
	}{
		{"test_1", "unibus/v2/filtered/ parrot in/ GARDden", "unibus/v2/",
			[]string{"filtered", "%20parrot%20in", "%GARDden"}},
		{"test_2", "unibus/v2/", "/v2/",
			[]string{"unibus", "v2"}},
		{"test_3", "unibus//", "unibus",
			[]string{}},
		{"test_4", "unibus/ /", "unibus",
			[]string{"%20"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, runFragmentsTest(tt.path, tt.rootPath, tt.expected))
	}
}

// TestGetDomainStatus tests the returned status messages. Error messages are not tested for.
func TestGetDomainStatus(t *testing.T) {
	testCases := []struct {
		name         string
		URL          string
		expected     string
		timeout      bool
		shouldLogErr bool
	}{
		{
			name:         "Successful request",
			URL:          "https://www.google.com",
			expected:     "200 OK",
			timeout:      false,
			shouldLogErr: false,
		},
		{
			name:         "Timeout error",
			URL:          "https://www.slowserver.com",
			expected:     "408 Request Timeout",
			timeout:      true,
			shouldLogErr: true,
		},
		{
			name:         "Protocol error",
			URL:          "https://www.notarealwebsite12345.com",
			expected:     "503 Service Unavailable",
			timeout:      false,
			shouldLogErr: true,
		},
	}
	// defines a slice of test structs before running the above function on each struct.
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			st, _ := util.GetDomainStatus(tc.URL)
			if tc.expected != st {
				t.Errorf("expected status %v but got %v", tc.expected, st)
			}
		})
	}
}

func TestEncodeAndWriteResponse(t *testing.T) {
	runEncodeTest := func(expected int, object interface{}) func(t *testing.T) {
		return func(t *testing.T) {
			handler := func(w http.ResponseWriter) {
				util.EncodeAndWriteResponse(&w, object)
			}
			w := httptest.NewRecorder()
			handler(w)
			resp := w.Result()
			if resp.StatusCode != expected {
				t.Errorf("Expected status #{expected} but got #{resp.StatusCode}")
			}
		}
	}
	testCases := []struct {
		Name             string
		ExpectedResponse int
		Object           interface{}
	}{
		{"Valid encode test",
			200,
			"This is a test encoding"},
		{"Invalid encode test",
			500,
			make(chan int)},
	}
	for _, test := range testCases {
		t.Run(test.Name,
			runEncodeTest(test.ExpectedResponse, test.Object))

	}

}

func TestHasCountryInRecords(t *testing.T) {
	var dataset util.CountryDataset
	err := dataset.Initialize(consts.DataSetPath)
	if err != nil {
		t.Error(err)
	}
	assert.True(t, dataset.HasCountryInRecords("NOR"))
	assert.True(t, dataset.HasCountryInRecords("Norway"))
	assert.False(t, dataset.HasCountryInRecords("E"))
	assert.False(t, dataset.HasCountryInRecords("NOTVALID"))
	assert.False(t, dataset.HasCountryInRecords(""))
}

/*
// HandleOutgoing takes a HandlerContext, request method, target url and a reader object along with a pointer
// to an object to be used for decoding.
// target point to an object expected to match the returned json body resulting from the request.
// returns:
// On failure: a string detailing in what step the error occurred for use with logging, along with error object
// On success: "", nil
func HandleOutgoing(handler *HandlerContext, method string, URL string, reader io.Reader, target interface{}) (string, error) {
	request, err := http.NewRequest(method, URL, reader)
	if err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "failed to create request", err
	}

	response, err := handler.Client.Do(request)
	if err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "request to" + request.URL.Path + " failed", err
	}

	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(target); err != nil {
		http.Error(*handler.Writer, "", http.StatusInternalServerError)
		return handler.Name + "failed to decode", err
	}
	return "", nil
}

*/
