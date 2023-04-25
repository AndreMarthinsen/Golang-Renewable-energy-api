package testing

import (
	"Assignment2/consts"
	"Assignment2/util"
	"github.com/stretchr/testify/assert"
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

	testConfig = util.Config{}
	testConfig.Initialize(consts.ConfigPath)
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
			expected:     "timed out contacting service",
			timeout:      true,
			shouldLogErr: true,
		},
		{
			name:         "Protocol error",
			URL:          "https://www.notarealwebsite12345.com",
			expected:     "protocol error",
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

/*
// TODO: Implement tests for the ones below, but verify that they are in use first and possibly refactor beforehand.
// EncodeAndWriteResponse attempts to encode data as a json response. Data must be a pointer
// to an appropriate object suited for encoding as json. Logging and errors are handled
// within the function.
func EncodeAndWriteResponse(w *http.ResponseWriter, data interface{}) {
	encoder := json.NewEncoder(*w)
	if err := encoder.Encode(data); err != nil {
		log.Println("Encoding error:", err)
		http.Error(*w, "Error during encoding", http.StatusInternalServerError)
		return
	}
	http.Error(*w, "", http.StatusOK)
}

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
