package firebase

import (
	"Assignment2/consts"
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// RequestStatus represents a http status code
// TODO: Any point to this?
type RequestStatus int16

// CacheResponse maps requested codes to resulting neighbours
// along with a http status code associated with any outgoing
// request to fetch the information.
type CacheResponse struct {
	Neighbours map[string][]string
	Status     RequestStatus
}

// CacheRequest wraps a pointer to a channel where the response
// should be posted along with a slice of country codes to be
// looked up in cache or external API
type CacheRequest struct {
	ChannelPtr     chan CacheResponse
	CountryRequest []string
}

// CacheEntry contains information about the borders of a country,
// its cca3 code and the time it was last updated.
type CacheEntry struct {
	Borders     []string  `firestore:"borders"`
	Cca3        string    `firestore:"cca3"`
	LastUpdated time.Time `firestore:"timestamp"`
}

// Config contains project config. TODO: Moves this to a more fitting package.
type Config struct {
	CachePushRate     float64
	CacheTimeLimit    time.Duration
	DebugMode         bool
	DevelopmentMode   bool
	Ctx               *context.Context
	FirestoreClient   *firestore.Client
	CachingCollection string
	PrimaryCache      string
}

type CacheMiss struct {
	Request  CacheRequest
	Response CacheResponse
}

// RunCacheWorker runs a worker intended for the purpose of supplying handlers for country
// neighbour data from in memory cache that is kept synced with external DB.
func RunCacheWorker(cfg *Config, requests <-chan CacheRequest, stop <-chan struct{},
	cleanupDone chan<- struct{}) {

	if cfg.DebugMode {
		log.Println("Cache worker: running")
	}
	localCache := make(map[string]CacheEntry, 0)
	cacheMisses := make([]CacheMiss, 0)
	client := http.Client{}

	localCache, err := loadCacheFromDB(cfg, cfg.PrimaryCache)
	if err != nil {
		log.Println("Cache worker: failed to load primary cache")
		log.Println("^ details: ", err)
	}
	if cfg.DebugMode {
		log.Println("Cache worker: local cache loaded with", len(localCache), "entries")
	}
	localCache, err = purgeStaleEntries(cfg, "PurgeTest", localCache)
	if err != nil {
		log.Println("Cache worker: failed to purge old entries")
		log.Println("^ details: ", err)
	}
	err = createCacheInDB(cfg, "TestStorage", localCache)
	if err != nil {
		log.Println("Cache worker: failed to create new cache file in DB")
		log.Println("^ details: ", err)
	}
	previousUpdate := time.Now()

	for {
		select { // Runs loop until it receives signal on stop channel
		case <-stop:
			if err = createCacheInDB(cfg, "ShutDownTest", localCache); err != nil {
				log.Fatal("cache worker: failed to create DB on shutdown")
			}
			cleanupDone <- struct{}{}
			return
		case val, ok := <-requests: // TODO: Limiting handled requests before considering default case
			if !ok {
				log.Println("Cache worker lost contact with request channel.\n" +
					"Running cleanup routine and shutting down cache worker.")
				cleanupDone <- struct{}{}
				return
			}
			// CHECKING AGAINST IN MEMORY CACHE //////////////////////////////
			response := CacheResponse{Status: http.StatusOK, Neighbours: map[string][]string{}}
			misses := make([]string, 0)
			for _, code := range val.CountryRequest {
				cacheResult, ok := localCache[code]
				if ok {
					response.Neighbours[code] = cacheResult.Borders
				} else {
					misses = append(misses, code)
				}
			}
			if len(misses) == 0 {
				if cfg.DebugMode {
					log.Println("returning response")
				}
				val.ChannelPtr <- response
			} else { // Some misses, will be handled when default case occurs
				val.CountryRequest = misses
				cacheMisses = append(cacheMisses, CacheMiss{Request: val, Response: response})
			}
		default: // SYNC OF DB, CHECKING THIRD PARTY API AGAINST MISSES /////////
			if len(cacheMisses) != 0 {
				updateLocalCache(cfg, &client, localCache, cacheMisses)
				for _, miss := range cacheMisses {
					for _, code := range miss.Request.CountryRequest {
						if cacheResult, ok := localCache[code]; ok {
							miss.Response.Neighbours[code] = cacheResult.Borders
						}
					}
					if len(miss.Response.Neighbours) == 0 {
						miss.Response.Status = http.StatusNotFound
					} // Cache updated and response sent to handler
					miss.Request.ChannelPtr <- miss.Response
				}
				cacheMisses = make([]CacheMiss, 0) // resets list over misses
			}
			if time.Since(previousUpdate).Seconds() >= cfg.CachePushRate {
				//TODO: Update external DB
				previousUpdate = time.Now()
			}
		}
	}
}

// updateLocalCache updates the local cache by attempting to retrieve data matching
// any registered misses. Requests are either made to internal stubbing if development is
// set in config, 3d party API if false.
func updateLocalCache(cfg *Config, client *http.Client, cache map[string]CacheEntry, misses []CacheMiss) {
	joinedCountryCodes := getCodesStringFromMisses(misses)
	var url string
	if cfg.DevelopmentMode { // Uses internal stubbing service when in development mode
		url = consts.StubDomain + consts.CountryCodePath + "?codes=" + joinedCountryCodes
	} else {
		url = consts.StubDomain + consts.CountryCodePath + joinedCountryCodes
	}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("Cache Worker failed to create request to url " + url)
	}
	response, err2 := client.Do(request)
	if err2 != nil {
		log.Println("cache worker failed to do request with url" + url)
	}

	returnedData := make([]CacheEntry, 0)
	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(&returnedData); err == nil {
		// Update of cache with any valid results
		for _, data := range returnedData {
			cache[data.Cca3] = data
		}
	}
}

// getCodesStringFromMisses collects all cca3 codes from the cache misses and
// concatenates them into a single string of codes separated by ','
func getCodesStringFromMisses(misses []CacheMiss) string {
	missedCodes := make(map[string]bool)
	// map is used to create a set of unique values. Bool is there to have a val per key.
	for _, miss := range misses {
		for _, code := range miss.Request.CountryRequest {
			missedCodes[code] = true
		}
	}
	var countryCodes = make([]string, 0, len(missedCodes))
	for code, _ := range missedCodes {
		countryCodes = append(countryCodes, code)
	}
	return strings.Join(countryCodes, ",")
}

// loadCacheFromDB loads a cache doc with the given ID from the collection
// determined by Config.CachingCollection.
// On success: (in mem cache as string -> CacheEntry map, nil)
// On fail:    (nil, error)
func loadCacheFromDB(cfg *Config, cacheID string) (map[string]CacheEntry, error) {
	res := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID)
	doc, err := res.Get(*cfg.Ctx)
	if err != nil {
		return nil, err
	}
	cacheMap := make(map[string]CacheEntry, 0)
	if err = doc.DataTo(&cacheMap); err != nil {
		return nil, err
	}
	return cacheMap, nil
}

// createCacheInDB creates a new cache in the external DB in a collection set by
// Config.CachingCollection with a given cacheID
// On success: nil
// On fail: error
func createCacheInDB(cfg *Config, cacheID string, cache map[string]CacheEntry) error {
	ref := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID)
	_, err := ref.Set(*cfg.Ctx, &cache)
	if err != nil {
		return err
	}
	return nil
}

// updateCacheInDB updates the remote cache with any new entries.
// TODO: Not currently functional
func updateCacheInDB(cfg *Config, cacheID string, newEntries map[string]CacheEntry) error {
	ref := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID)
	if _, err := ref.Set(*cfg.Ctx, newEntries, firestore.MergeAll); err != nil {
		return err
	}
	return nil
}

// purgeStaleEntries removes entries older than the time-limit set in Config.CacheTimeLimit
// from the local cache as well as the remote DB.
func purgeStaleEntries(cfg *Config, cacheID string, oldCache map[string]CacheEntry) (map[string]CacheEntry, error) {

	newCache := make(map[string]CacheEntry, 0)
	for key, val := range oldCache {
		if time.Since(val.LastUpdated) < cfg.CacheTimeLimit {
			val.LastUpdated = time.Now()
			newCache[key] = val
		}
	}
	ref := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID)
	_, err := ref.Set(*cfg.Ctx, newCache, firestore.MergeAll)
	if err != nil {
		return nil, err
	}
	oldCache = nil
	return newCache, nil
}
