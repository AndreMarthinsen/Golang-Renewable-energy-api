package caching

import (
	"Assignment2/fsutils"
	"Assignment2/util"
	"log"
	"net/http"
	"time"
)

// RunCacheWorker runs a worker intended for the purpose of supplying handlers for country
// neighbour data from in memory cache that is kept synced with external DB.
//
// The cache worker will run until the 'stop' channel is signaled on.
// Stopping the worker, or closing the 'requests' channel, will cause the worker to attempt
// doing a shut-down routine, synchronizing the local cache with the external DB before
// signaling on 'done' to signify that it has completed the shutdown.
func RunCacheWorker(cfg *util.Config, requests chan CacheRequest, stop <-chan struct{},
	cleanupDone chan<- struct{}) {

	if cfg.DebugMode {
		log.Println("Cache worker: running")
	}

	// slice with any cache misses that need handling
	cacheMisses := make([]CacheMiss, 0)
	client := http.Client{}
	cacheUpdated := false

	// map from cca3 codes to CacheEntry structs with borders and timestamp.
	localCache, err := localCacheInit(cfg)
	if err != nil {
		log.Println(err)
	}

	// Main request-handling loop. Runs until a stop signal is received or request channel is closed.
	for {
		select {
		case <-time.After(cfg.CachePushRate):
			if cacheUpdated {
				// Updates external Cache file by overwriting
				err := fsutils.AddDocumentById(cfg, cfg.CachingCollection, cfg.PrimaryCache, &localCache)
				if err != nil {
					log.Println("cache worker: failed to update cache in DB on periodic update")
				}
				cacheUpdated = false
			}
		case <-stop: // Signal received on stop channel, shutting down worker.
			// Writes to primary cache in db before shutting down
			err := fsutils.AddDocumentById(cfg, cfg.CachingCollection, cfg.PrimaryCache, &localCache)
			if err != nil {
				log.Println("cache worker: failed to create DB on shutdown")
			}
			cleanupDone <- struct{}{}
			return
		case probe, ok := <-requests: // Either request has been received or channel is closed
			if !ok {
				log.Println("Cache worker lost contact with request channel.\n" +
					"Running cleanup routine and shutting down cache worker.")
				err := fsutils.AddDocumentById(cfg, cfg.CachingCollection, cfg.PrimaryCache, &localCache)
				if err != nil {
					log.Println("cache worker: failed to create DB on shutdown")
				}
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
				cacheUpdated = updateLocalCache(cfg, &client, &localCache, cacheMisses) || cacheUpdated
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
