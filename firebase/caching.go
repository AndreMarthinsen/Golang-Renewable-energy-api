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

type RequestStatus int16

type CacheResponse struct {
	Neighbours map[string][]string
	Status     RequestStatus
}

type CacheRequest struct {
	ChannelPtr     *chan CacheResponse
	CountryRequest []string
}

type CountryBorder struct {
	Cca3    string   `json:"cca3"`
	Borders []string `json:"borders"`
}

type Cache struct {
	CacheEntries []CacheEntry `firestore:"root"`
}

type CacheEntry struct {
	Borders     []string  `firestore:"borders"`
	Cca3        string    `firestore:"cca3"`
	LastUpdated time.Time `firestore:"timestamp"`
}

func ConvertHitsToResponse(hits []CacheEntry) CacheResponse {
	response := CacheResponse{Status: http.StatusOK} // TODO: What sort of information are we interested in returning?
	for _, hit := range hits {
		response.Neighbours[hit.Cca3] = hit.Borders
	}
	if len(response.Neighbours) == 0 {
		response.Status = http.StatusNotFound
	}
	return response
}

func (entry *CacheEntry) toCountryBorder() CountryBorder {
	return CountryBorder{Borders: entry.Borders, Cca3: entry.Cca3}
}

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
			response := CacheResponse{Status: http.StatusOK}
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
				*val.ChannelPtr <- response
			} else { // Some misses, will be handled when default case occurs
				val.CountryRequest = misses
				cacheMisses = append(cacheMisses, CacheMiss{Request: val, Response: response})
			}
		default: // SYNC OF DB, CHECKING THIRD PARTY API AGAINST MISSES /////////
			if len(cacheMisses) != 0 {
				updateLocalCache(&client, localCache, cacheMisses)
				for _, miss := range cacheMisses {
					for _, code := range miss.Request.CountryRequest {
						if cacheResult, ok := localCache[code]; ok {
							miss.Response.Neighbours[code] = cacheResult.Borders
						}
					}
					if len(miss.Response.Neighbours) == 0 {
						miss.Response.Status = http.StatusNotFound
					} // Cache updated and response sent to handler
					*miss.Request.ChannelPtr <- miss.Response
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

func updateLocalCache(client *http.Client, cache map[string]CacheEntry, misses []CacheMiss) {
	joinedCountryCodes := getCodesStringFromMisses(misses)
	request, err := http.NewRequest(
		http.MethodGet,
		consts.StubDomain+consts.CountryCodePath+joinedCountryCodes,
		nil,
	)
	if err != nil {
		log.Println("Cache Worker failed to create request.")
	}
	response, err2 := client.Do(request)
	if err2 != nil {
		log.Println("cache worked failed to do request")
	}

	returnedData := make([]CacheEntry, 0)
	decoder := json.NewDecoder(response.Body)
	if err = decoder.Decode(&returnedData); err != nil {
		log.Println("failed to decode")
	} else {
		// Update of cache with any valid results
		for _, data := range returnedData {
			cache[data.Cca3] = data
		}
	}
}

func getCodesStringFromMisses(misses []CacheMiss) string {
	codes := make([]string, 0)
	for _, miss := range misses {
		codes = append(codes, miss.Request.CountryRequest...)
	}
	missedCountries := make(map[string]int8) // int serves no purpose,
	for _, code := range codes {             // map is just used to create a set of unique values
		missedCountries[code] = 0
	}
	countryCodes := make([]string, 0)
	for key, _ := range missedCountries {
		countryCodes = append(countryCodes, key)
	}
	return strings.Join(countryCodes, ",")
}

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

func createCacheInDB(cfg *Config, cacheID string, cache map[string]CacheEntry) error {
	ref := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID)
	_, err := ref.Set(*cfg.Ctx, &cache)
	if err != nil {
		return err
	}
	return nil
}

func updateCacheInDB(cfg *Config, cacheID string, newEntries map[string]CacheEntry) error {
	ref := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID)
	if _, err := ref.Set(*cfg.Ctx, newEntries, firestore.MergeAll); err != nil {
		return err
	}
	return nil
}

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
