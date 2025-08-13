package main

import (
	"testing"
	"time"

	"github.com/Thijs-Desjardijn/pokedex/internal/pokecache"
)

func TestGetData_CacheHitAndMiss(t *testing.T) {
	cache := pokecache.NewCache(5 * time.Second)
	testURL := "https://pokeapi.co/api/v2/location-area/1"
	fakeResp := []byte(`{"field":"value"}`)

	// Test: put data in cache, should be a cache hit
	cache.Add(testURL, fakeResp)
	out, err := GetData(cache, testURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != string(fakeResp) {
		t.Errorf("cache hit expected %q, got %q", fakeResp, out)
	}

	// Test: request a real API URL (not yet in cache)
	liveURL := "https://pokeapi.co/api/v2/location-area/2"
	out2, err := GetData(cache, liveURL)
	if err != nil {
		t.Fatalf("unexpected error for live API: %v", err)
	}
	if len(out2) == 0 {
		t.Error("expected non-empty response from PokeAPI")
	}
	// After the call, should now be in the cache
	out3, ok := cache.Get(liveURL)
	if !ok || string(out3) != string(out2) {
		t.Error("expected live API response to be cached")
	}
}
