package caching

import "time"

// RequestStatus represents a http status code
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

// CacheMiss wraps the so far built up response and a modified CacheRequest containing
// only the cca3 codes resulting in a cache miss.
type CacheMiss struct {
	Request  CacheRequest
	Response CacheResponse
}
