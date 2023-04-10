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
	Neighbours CountryBorder
	Status     RequestStatus
}

type CacheRequest struct {
	ChannelPtr     *chan CacheResponse
	CountryRequest string
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

func (entry *CacheEntry) toCountryBorder() CountryBorder {
	return CountryBorder{Borders: entry.Borders, Cca3: entry.Cca3}
}

type Config struct {
	CachePushRate     float64
	DebugMode         bool
	DevelopmentMode   bool
	Ctx               *context.Context
	FirestoreClient   *firestore.Client
	CachingCollection string
	PrimaryCache      string
}

func RunCacheWorker(cfg *Config, requests <-chan CacheRequest, stop <-chan struct{},
	cleanupDone chan<- struct{}) {
	if cfg.DebugMode {
		log.Println("Cache worker: running")
	}
	localCache := make(map[string]CacheEntry, 0)
	cacheMisses := make([]CacheRequest, 0)
	client := http.Client{}

	updateDB := func() {}
	localCache, err := loadCacheFromDB(cfg, cfg.PrimaryCache)
	if err != nil {
		log.Println("Cache worker: failed to load primary cache")
		log.Println("^ details:", err)
	}
	if cfg.DebugMode {
		log.Println("Cache worker: local cache loaded with", len(localCache), "entries")
	}
	err = createCacheInDB(cfg, "TestStorage", localCache)
	if err != nil {
		log.Println("Cache worker: failed to create new cache file in DB")
		log.Println("^ details:", err)
	}
	previousUpdate := time.Now()

	for {
		select { // Runs loop until it receives signal on stop channel
		case <-stop:
			if err := updateCacheInDB(cfg, "", localCache); err != nil {
				//TODO: Error handling goes here
			}
			cleanupDone <- struct{}{}
			return
		default:
		} // Updates external DB with a given interval
		if time.Since(previousUpdate).Seconds() >= cfg.CachePushRate {
			updateDB()
			previousUpdate = time.Now()
		}
		for {
			select {
			case val, ok := <-requests:
				if !ok {
					log.Println("Cache worker lost contact with request channel.\n" +
						"Running cleanup routine and shutting down cache worker.")
					updateDB()
					cleanupDone <- struct{}{}
					return
				}
				cacheResult, ok := localCache[val.CountryRequest]
				if ok {
					*val.ChannelPtr <- CacheResponse{
						cacheResult.toCountryBorder(),
						http.StatusOK,
					}
				} else {
					cacheMisses = append(cacheMisses, val)
				}
			default:
				if len(cacheMisses) != 0 {
					updateLocalCache(&client, localCache, cacheMisses)
					for _, miss := range cacheMisses {
						if cacheResult, ok := localCache[miss.CountryRequest]; ok {
							*miss.ChannelPtr <- CacheResponse{
								cacheResult.toCountryBorder(),
								http.StatusOK,
							}
						} else {
							*miss.ChannelPtr <- CacheResponse{
								CountryBorder{},
								http.StatusBadRequest,
							} // TODO: Probably a better way to handle this.
						}
					}
					cacheMisses = make([]CacheRequest, 0) // resets list over misses
					break
				}
			}
		}
	}
}

func updateLocalCache(client *http.Client, cache map[string]CacheEntry, misses []CacheRequest) {
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

func getCodesStringFromMisses(misses []CacheRequest) string {
	missedCountries := make(map[string]int8) // int serves no purpose,
	for _, miss := range misses {            // map is just used to create a set of unique values
		missedCountries[miss.CountryRequest] = 0
	}
	countryCodes := make([]string, 0)
	for key, _ := range missedCountries {
		countryCodes = append(countryCodes, key)
	}
	return strings.Join(countryCodes, ",")
}

func loadCacheFromDB(cfg *Config, CacheID string) (map[string]CacheEntry, error) {
	res := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(CacheID)
	doc, err := res.Get(*cfg.Ctx)
	if err != nil {
		return nil, err
	}
	cache := Cache{}
	if err = doc.DataTo(&cache); err != nil {
		return nil, err
	}
	cacheMap := make(map[string]CacheEntry, 0)
	for _, entry := range cache.CacheEntries {
		cacheMap[entry.Cca3] = entry
	}
	return cacheMap, nil
}

func createCacheInDB(cfg *Config, cacheID string, cache map[string]CacheEntry) error {
	c := Cache{}
	for _, val := range cache {
		c.CacheEntries = append(c.CacheEntries, val)
	}
	res, _, err := cfg.FirestoreClient.Collection(cfg.CachingCollection).Add(*cfg.Ctx, cacheID)
	if err != nil {
		return err
	}
	if _, err = res.Set(*cfg.Ctx, &c); err != nil {
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

func purgeStaleEntries(cfg *Config, cacheID string, oldCache map[string]CacheEntry, timeLimit time.Duration) (map[string]CacheEntry, error) {
	newCache := make(map[string]CacheEntry, 0)
	for key, val := range oldCache {
		if time.Since(val.LastUpdated) < timeLimit {
			val.LastUpdated = time.Now()
			newCache[key] = val
		}
	}
	_, err := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID).Set(*cfg.Ctx, newCache, firestore.MergeAll)
	if err != nil {
		return nil, err
	}
	oldCache = nil
	return newCache, nil
}
