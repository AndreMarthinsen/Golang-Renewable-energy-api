package util

import (
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"time"
)

const SettingsCachePushRate = 5 * time.Second
const SettingsCacheTimeLimit = 1 * time.Hour
const SettingsWebhookEventRate = 10 * time.Second
const SettingsDebugMode = true
const SettingsDevelopmentMode = true
const SettingsCachingCollection = "Caches"
const SettingsPrimaryCache = "TestData"
const SettingsWebhookCollection = "Webhooks"

const minimumWebhookInterval = 10

// Config contains project config.
type Config struct {
	CachePushRate     time.Duration // Cache is pushed to external DB with CachePushRate as its interval
	CacheTimeLimit    time.Duration // Cache entries older than CacheTimeLimit are purged upon loading
	WebhookEventRate  time.Duration // How often registered webhooks should be checked for event triggers
	DebugMode         bool          // toggles any extra debug features such as extra logging of events
	DevelopmentMode   bool          // Sets the service to use stubbing of external APIs
	Ctx               *context.Context
	FirestoreClient   *firestore.Client
	CachingCollection string
	PrimaryCache      string
	WebhookCollection string
}

// configYAML is used to decode the settings from the project config.yaml file.
type configYAML struct {
	Intervals struct {
		CachePushRate    int `yaml:"cache-push-rate"`
		CacheTimeLimit   int `yaml:"cache-time-limit"`
		WebhookEventRate int `yaml:"webhook-event-rate"`
	} `yaml:"time-intervals"`

	Deployment struct {
		DebugMode       bool `yaml:"debug-mode"`
		DevelopmentMode bool `yaml:"development-mode"`
	} `yaml:"deployment-variables"`

	Firebase struct {
		CachingCollectionName    string `yaml:"caching-collection-name"`
		PrimaryCacheDocumentName string `yaml:"primary-cache-document-name"`
		WebhookCollectionName    string `yaml:"webhook-collection-name"`
	} `yaml:"firebase-variables"`
}

// InitializeWithDefaults sets config settings to their defaults.
func (c *Config) InitializeWithDefaults() {
	c.CachePushRate = SettingsCachePushRate
	c.CacheTimeLimit = SettingsCacheTimeLimit
	c.DebugMode = SettingsDebugMode
	c.DevelopmentMode = SettingsDevelopmentMode
	c.CachingCollection = SettingsCachingCollection
	c.PrimaryCache = SettingsPrimaryCache
	c.WebhookCollection = SettingsWebhookCollection
	c.WebhookEventRate = SettingsWebhookEventRate
}

// Initialize resets config settings to their defaults by calling InitializeWithDefaults
// before attempting to parse settings from the project config file.
func (c *Config) Initialize(path string) {
	configData, err := os.ReadFile(path)
	if err != nil {
		log.Println("server config: failed to load configuration, running with default settings.", err)
	}
	reader := bytes.NewReader(configData)
	decoder := yaml.NewDecoder(reader)
	// Set the custom encoder and decoder functions for duration values.
	c.InitializeWithDefaults()

	temp := configYAML{}
	if err := decoder.Decode(&temp); err != nil {
		log.Println("server config: failed to load configuration, running with default settings.", err)
	}

	// Sets non-default time intervals only if non-zero or above set limitations.
	if temp.Intervals.CachePushRate != 0 {
		c.CachePushRate = time.Duration(temp.Intervals.CachePushRate) * time.Second
	}
	if temp.Intervals.CacheTimeLimit != 0 {
		c.CacheTimeLimit = time.Duration(temp.Intervals.CacheTimeLimit) * time.Minute
	}
	if temp.Intervals.WebhookEventRate > minimumWebhookInterval {
		c.WebhookEventRate = time.Duration(temp.Intervals.WebhookEventRate) * time.Second
	}
	// copy of remaining fields.
	c.DebugMode = temp.Deployment.DebugMode
	c.DevelopmentMode = temp.Deployment.DevelopmentMode
	c.CachingCollection = temp.Firebase.CachingCollectionName
	c.PrimaryCache = temp.Firebase.PrimaryCacheDocumentName
	c.WebhookCollection = temp.Firebase.WebhookCollectionName
}
