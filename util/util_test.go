package util

import (
	"Assignment2/consts"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigInitialize(t *testing.T) {
	var testConfig Config

	testConfig.InitializeWithDefaults()
	defaultConfig := Config{
		CachePushRate:     SettingsCachePushRate,
		CacheTimeLimit:    SettingsCacheTimeLimit,
		DebugMode:         SettingsDebugMode,
		DevelopmentMode:   SettingsDevelopmentMode,
		CachingCollection: SettingsCachingCollection,
		PrimaryCache:      SettingsPrimaryCache,
		WebhookCollection: SettingsWebhookCollection,
		WebhookEventRate:  SettingsWebhookEventRate,
	}
	assert.Equal(t, defaultConfig, testConfig)
	assert.Nil(t, testConfig.Initialize("../config/config.yaml"))
	assert.Error(t, testConfig.Initialize("/invalid_path"))
}

func TestStatusToText(t *testing.T) {
	assert.Equal(t, "200 OK", StatusToString(http.StatusOK))
	assert.Equal(t, "404 Not Found", StatusToString(http.StatusNotFound))
	assert.Equal(t, "", StatusToString(3000))
}

func TestSetUpServiceConfig(t *testing.T) {
	panicTest := func(cfg *Config) func() {
		return func() {
			// panic on nil pointer dereference
			err := cfg.FirestoreClient.Close()
			if err != nil {
				t.Error(err)
			}
		}
	}

	config, err := SetUpServiceConfig("/invalid/path", "../cmd/sha.json")
	assert.Nil(t, err)
	assert.NotPanics(t, panicTest(&config), config)

	config, err = SetUpServiceConfig("."+consts.ConfigPath, "./invalid_creds.json")
	assert.Error(t, err)
	assert.Panics(t, panicTest(&config), config)

	config, err = SetUpServiceConfig("."+consts.ConfigPath, "../cmd/sha.json")
	assert.Nil(t, err)
	assert.NotPanics(t, panicTest(&config))

}

func TestMax(t *testing.T) {
	assert.Equal(t, 5, Max(5, 0))
	assert.Equal(t, 0, Max(-5, 0))
	assert.Equal(t, 0.0, Max(0.0, 0.0))
	assert.Equal(t, 'b', Max('b', 'a'))
}

func TestMin(t *testing.T) {
	assert.Equal(t, 0, Min(5, 0))
	assert.Equal(t, -5, Min(-5, 0))
	assert.Equal(t, 0.0, Min(0.0, 0.0))
	assert.Equal(t, 'a', Min('b', 'a'))
}

// TestFragmentsFromPath test that verifies parsing of an url string into a set of segments.
func TestFragmentsFromPath(t *testing.T) {
	runFragmentsTest := func(path string, rootPath string, expected []string) func(t *testing.T) {
		return func(t *testing.T) {
			processed := FragmentsFromPath(path, rootPath)
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
			st, _ := GetDomainStatus(tc.URL)
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
				EncodeAndWriteResponse(&w, object)
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
	var dataset CountryDataset
	err := dataset.Initialize("." + consts.DataSetPath)
	if err != nil {
		t.Error(err)
	}
	assert.True(t, dataset.HasCountryInRecords("NOR"))
	assert.False(t, dataset.HasCountryInRecords("E"))
	assert.False(t, dataset.HasCountryInRecords("NOTVALID"))
	assert.False(t, dataset.HasCountryInRecords(""))
}
