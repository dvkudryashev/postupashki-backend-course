package main

import (
	"strings"
	"time"
)

type CacheKey struct {
	Name string
	Type DNSType
}

type CacheEntry struct {
	Answers   []DNSAnswer
	ExpiresAt time.Time
}

func makeCacheKey(name string, queryType DNSType) CacheKey {
	return CacheKey{
		Name: strings.ToLower(name),
		Type: queryType,
	}
}

func getCached(cache map[CacheKey]CacheEntry, key CacheKey, now time.Time) (CacheEntry, bool) {
	entry, exists := cache[key]

	if !exists {
		return CacheEntry{}, false
	}

	if now.After(entry.ExpiresAt) || now.Equal(entry.ExpiresAt) {
		delete(cache, key)
		return CacheEntry{}, false
	}

	return entry, true
}

func putCache(cache map[CacheKey]CacheEntry, key CacheKey, response DNSResponse, now time.Time) {
	if response.Status != NOERROR {
		return
	}

	if response.AnswerCount == 0 {
		return
	}

	if response.MinTTL == 0 {
		return
	}

	expiresAt := now.Add(time.Duration(response.MinTTL) * time.Second)

	cache[key] = CacheEntry{
		Answers:   response.Answers,
		ExpiresAt: expiresAt,
	}
}
