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
	ChannelRef     chan CacheResponse
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
	CachePushRate     time.Duration
	CacheTimeLimit    time.Duration
	DebugMode         bool
	DevelopmentMode   bool
	Ctx               *context.Context
	FirestoreClient   *firestore.Client
	CachingCollection string
	PrimaryCache      string
}

// CacheMiss wraps the so far built up response and a modified CacheRequest containing
// only the cca3 codes resulting in a cache miss.
type CacheMiss struct {
	Request  CacheRequest
	Response CacheResponse
}

// RunCacheWorker runs a worker intended for the purpose of supplying handlers for country
// neighbour data from in memory cache that is kept synced with external DB.
//
// The cache worker will run until the 'stop' channel is signaled on.
// Stopping the worker, or closing the 'requests' channel, will cause the worker to attempt
// doing a shut-down routine, synchronizing the local cache with the external DB before
// signaling on 'done' to signify that it has completed the shutdown.
func RunCacheWorker(cfg *Config, requests chan CacheRequest, stop <-chan struct{},
	cleanupDone chan<- struct{}) {

	if cfg.DebugMode {
		log.Println("Cache worker: running")
	}

	// slice with any cache misses that need handling
	cacheMisses := make([]CacheMiss, 0)
	client := http.Client{}
	cacheUpdated := false

	// map from cca3 codes to CacheEntry structs with borders and timestamp.
	localCache := localCacheInit(cfg)

	// Main request-handling loop. Runs until a stop signal is received or request channel is closed.
	for {
		select {
		case <-time.After(time.Second * 5):
			if cacheUpdated {
				if err := updateCacheInDB(cfg, "UpdatedCache", localCache); err != nil {
					log.Println("cache worker: failed to update cache in DB on periodic update")
				}
				cacheUpdated = false
			}
		case <-stop: // Signal received on stop channel, shutting down worker.
			if err := updateCacheInDB(cfg, "ShutDownTest", localCache); err != nil {
				log.Fatal("cache worker: failed to create DB on shutdown")
			}
			cleanupDone <- struct{}{}
			return
		case probe, ok := <-requests: // Either request has been received or channel closed
			if !ok { // TODO: Limiting handled requests before considering default case
				log.Println("Cache worker lost contact with request channel.\n" +
					"Running cleanup routine and shutting down cache worker.")
				cleanupDone <- struct{}{}
				return
			}
			requests <- probe // value is put back into channel
			notDone := true
			for notDone {
				select {
				case val, _ := <-requests:
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
						val.ChannelRef <- response
					} else { // Some misses, will be handled when default case occurs
						val.CountryRequest = misses
						cacheMisses = append(cacheMisses, CacheMiss{Request: val, Response: response})
					}
				default:
					notDone = false
				}
			}
			if len(cacheMisses) != 0 {
				// Any cache misses are checked against the external api.
				// Any valid results are added to the local cache.
				temp := updateLocalCache(cfg, &client, &localCache, cacheMisses)
				cacheUpdated = cacheUpdated || temp
				// Iterates through the misses where each miss represents one Request
				for _, miss := range cacheMisses {
					// Iterates through the missed cca3 codes that were missed
					// and updates the CacheResponse with any hits in updated cache.
					for _, code := range miss.Request.CountryRequest {
						if cacheResult, ok := localCache[code]; ok {
							miss.Response.Neighbours[code] = cacheResult.Borders
						}
					}
					if len(miss.Response.Neighbours) == 0 {
						miss.Response.Status = http.StatusNotFound
					}
					// A final response sent to the handler that made the current
					miss.Request.ChannelRef <- miss.Response
				}
				// resets cache misses
				cacheMisses = make([]CacheMiss, 0)
			}
		}
	}
}

// localCacheInit initializes the local cache from the external DB, backs the ache up,
// and purges any outdated entries from the cache before returning it.
func localCacheInit(cfg *Config) map[string]CacheEntry {
	localCache, err := loadCacheFromDB(cfg, cfg.PrimaryCache)
	if err != nil {
		log.Println("cache worker: failed to load primary cache")
		log.Println("^ details: ", err)
	} else {
		if err = createCacheInDB(cfg, cfg.PrimaryCache+".backup", localCache); err != nil {
			log.Println("cache worker: failed to create backup of old cache")
			log.Println("^ details: ", err)
		}
	}
	if cfg.DebugMode {
		log.Println("cache worker: loaded local cache with", len(localCache), "entries")
	}
	localCache, err = purgeStaleEntries(cfg, "PurgeTest", localCache)
	if err != nil {
		log.Println("cache worker: failed to purge old entries")
		log.Println("^ details: ", err)
	}
	return localCache
}

// updateLocalCache updates the local cache by attempting to retrieve data matching
// any registered misses. Requests are either made to internal stubbing if development is
// set in config, 3d party API if false.
func updateLocalCache(cfg *Config, client *http.Client, cache *map[string]CacheEntry, misses []CacheMiss) bool {
	joinedCountryCodes := getCodesStringFromMisses(misses)
	var url string
	if cfg.DevelopmentMode { // Uses internal stubbing service when in development mode
		url = consts.StubDomain + consts.CountryCodePath + "?codes=" + joinedCountryCodes
	} else {
		url = consts.CountryDomain + consts.CountryCodePath + "?codes=" + joinedCountryCodes
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
			(*cache)[data.Cca3] = data
		}
		return true
	}
	return false
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
	for code := range missedCodes {
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
	if _, err := ref.Set(*cfg.Ctx, &newEntries); err != nil {
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
