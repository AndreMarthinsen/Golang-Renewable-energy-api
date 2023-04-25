package caching

import (
	"Assignment2/consts"
	"Assignment2/fsutils"
	"Assignment2/util"
	"cloud.google.com/go/firestore"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

// localCacheInit initializes the local cache from the external DB, backs the ache up,
// and purges any outdated entries from the cache before returning it.
//
// Failing to load the cache from the DB results in an empty map being returned.
func localCacheInit(cfg *util.Config) (map[string]CacheEntry, error) {
	localCache, err := loadCacheFromDB(cfg, cfg.PrimaryCache)
	if err != nil {
		localCache = make(map[string]CacheEntry, 0)
		return localCache, errors.New("cache worker: failed to load primary cache: " + err.Error())
	} else { // overwrites old backup file
		err = fsutils.AddDocumentById(cfg, cfg.CachingCollection, cfg.PrimaryCache+".backup", &localCache)
		if err != nil {
			return localCache, errors.New("cache worker: failed to create backup of old cache: " + err.Error())
		}
	}
	localCache, err = purgeStaleEntries(cfg, cfg.PrimaryCache, localCache)
	if err != nil {
		return nil, errors.New("cache worker: failed to purge old entries: " + err.Error())
	}
	return localCache, nil
}

// updateLocalCache updates the local cache by attempting to retrieve data matching
// any registered misses. Requests are either made to internal stubbing if development is
// set in config, 3d party API if false.
// Returns: True if cache has been updated, False otherwise.
func updateLocalCache(cfg *util.Config, client *http.Client, cache *map[string]CacheEntry, misses []CacheMiss) bool {
	joinedCountryCodes := getCodesStringFromMisses(misses)
	var url string
	if cfg.DevelopmentMode { // Uses internal stubbing service when in development mode
		url = consts.StubDomain
	} else {
		url = consts.CountryDomain
	}
	url += "//" + "?codes=" + joinedCountryCodes

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("Cache Worker failed to create request to url " + url)
		return false
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
			data.LastUpdated = time.Now()
			(*cache)[data.Cca3] = data
		}
		return true
	}
	return false
}

// getCodesStringFromMisses collects all cca3 codes from the cache misses and
// concatenates them into a single string of codes separated by ','
func getCodesStringFromMisses(misses []CacheMiss) string {
	uniqueCountryCodes := make(map[string]struct{})
	// map is used to create a set of unique values.
	for _, miss := range misses {
		for _, code := range miss.Request.CountryRequest {
			uniqueCountryCodes[code] = struct{}{}
		}
	}
	var countryCodes = make([]string, 0, len(uniqueCountryCodes))
	for code := range uniqueCountryCodes {
		countryCodes = append(countryCodes, code)
	}
	return strings.Join(countryCodes, ",")
}

// loadCacheFromDB loads a cache doc with the given ID from the collection
// determined by Config.CachingCollection.
// On success: (in mem cache as string -> CacheEntry map, nil)
// On fail:    (nil, error)
func loadCacheFromDB(cfg *util.Config, cacheID string) (map[string]CacheEntry, error) {
	cacheMap := make(map[string]CacheEntry, 0)
	err := fsutils.ReadDocumentGeneral(cfg, cfg.CachingCollection, cacheID, &cacheMap)
	return cacheMap, err
}

// purgeStaleEntries removes entries older than the time-limit set in Config.CacheTimeLimit
// from the local cache as well as the remote DB.
func purgeStaleEntries(cfg *util.Config, cacheID string, oldCache map[string]CacheEntry) (map[string]CacheEntry, error) {

	newCache := make(map[string]CacheEntry, 0)
	for key, val := range oldCache {
		if time.Since(val.LastUpdated) < cfg.CacheTimeLimit {
			val.LastUpdated = time.Now()
			newCache[key] = val
		}
	}
	ref := cfg.FirestoreClient.Collection(cfg.CachingCollection).Doc(cacheID)
	_, err := ref.Set(*cfg.Ctx, newCache, firestore.MergeAll)
	return newCache, err
}
