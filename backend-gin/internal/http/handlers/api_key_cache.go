package handlers

import (
	"container/list"
	"strconv"
	"sync"
	"time"
)

const (
	openAPIKeyScope = "open"
	userAPIKeyScope = "user"

	apiKeyPositiveCacheCapacity = 4096
	apiKeyNegativeCacheCapacity = 1024
	apiKeyLimiterCapacity       = 2048
	apiKeyTouchCapacity         = 4096

	apiKeyPositiveCacheTTL = 45 * time.Second
	apiKeyNegativeCacheTTL = 10 * time.Second
	apiKeyInvalidWindow    = time.Minute
	apiKeyInvalidLimit     = 30
	apiKeyInvalidBlockTTL  = 5 * time.Minute
	apiKeyTouchInterval    = 10 * time.Minute
)

type apiKeyCacheState uint8

const (
	apiKeyCacheMiss apiKeyCacheState = iota
	apiKeyCachePositive
	apiKeyCacheNegative
)

type apiKeyCacheEntry struct {
	key       string
	identity  int64
	value     any
	expiresAt time.Time
}

type apiKeyLRU struct {
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

func newAPIKeyLRU(capacity int) apiKeyLRU {
	if capacity < 1 {
		capacity = 1
	}
	return apiKeyLRU{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

func (lru *apiKeyLRU) get(key string, now time.Time) (apiKeyCacheEntry, bool) {
	element, ok := lru.items[key]
	if !ok {
		return apiKeyCacheEntry{}, false
	}
	entry := element.Value.(apiKeyCacheEntry)
	if !entry.expiresAt.IsZero() && !now.Before(entry.expiresAt) {
		lru.removeElement(element)
		return apiKeyCacheEntry{}, false
	}
	lru.order.MoveToFront(element)
	return entry, true
}

func (lru *apiKeyLRU) set(entry apiKeyCacheEntry) {
	if element, ok := lru.items[entry.key]; ok {
		element.Value = entry
		lru.order.MoveToFront(element)
		return
	}
	element := lru.order.PushFront(entry)
	lru.items[entry.key] = element
	for lru.order.Len() > lru.capacity {
		lru.removeElement(lru.order.Back())
	}
}

func (lru *apiKeyLRU) delete(key string) {
	if element, ok := lru.items[key]; ok {
		lru.removeElement(element)
	}
}

func (lru *apiKeyLRU) deleteIdentity(scope string, identity int64) {
	for element := lru.order.Front(); element != nil; {
		next := element.Next()
		entry := element.Value.(apiKeyCacheEntry)
		if entry.identity == identity && cacheKeyScope(entry.key) == scope {
			lru.removeElement(element)
		}
		element = next
	}
}

func (lru *apiKeyLRU) removeElement(element *list.Element) {
	if element == nil {
		return
	}
	entry := element.Value.(apiKeyCacheEntry)
	delete(lru.items, entry.key)
	lru.order.Remove(element)
}

type apiKeyLimiterEntry struct {
	clientID     string
	windowStart  time.Time
	count        int
	blockedUntil time.Time
}

type apiKeyTouchEntry struct {
	key        string
	reservedAt time.Time
}

type APIKeyAuthCache struct {
	mu sync.Mutex

	positive apiKeyLRU
	negative apiKeyLRU

	limiterCapacity int
	limiters        map[string]*list.Element
	limiterOrder    *list.List

	touchCapacity int
	touches       map[string]*list.Element
	touchOrder    *list.List

	positiveTTL   time.Duration
	negativeTTL   time.Duration
	invalidWindow time.Duration
	invalidLimit  int
	blockTTL      time.Duration
	touchInterval time.Duration
}

func NewAPIKeyAuthCache() *APIKeyAuthCache {
	return newAPIKeyAuthCache(
		apiKeyPositiveCacheCapacity,
		apiKeyNegativeCacheCapacity,
		apiKeyLimiterCapacity,
		apiKeyTouchCapacity,
		apiKeyPositiveCacheTTL,
		apiKeyNegativeCacheTTL,
		apiKeyInvalidWindow,
		apiKeyInvalidLimit,
		apiKeyInvalidBlockTTL,
		apiKeyTouchInterval,
	)
}

func newAPIKeyAuthCache(
	positiveCapacity int,
	negativeCapacity int,
	limiterCapacity int,
	touchCapacity int,
	positiveTTL time.Duration,
	negativeTTL time.Duration,
	invalidWindow time.Duration,
	invalidLimit int,
	blockTTL time.Duration,
	touchInterval time.Duration,
) *APIKeyAuthCache {
	if limiterCapacity < 1 {
		limiterCapacity = 1
	}
	if touchCapacity < 1 {
		touchCapacity = 1
	}
	if invalidLimit < 1 {
		invalidLimit = 1
	}
	return &APIKeyAuthCache{
		positive:        newAPIKeyLRU(positiveCapacity),
		negative:        newAPIKeyLRU(negativeCapacity),
		limiterCapacity: limiterCapacity,
		limiters:        make(map[string]*list.Element, limiterCapacity),
		limiterOrder:    list.New(),
		touchCapacity:   touchCapacity,
		touches:         make(map[string]*list.Element, touchCapacity),
		touchOrder:      list.New(),
		positiveTTL:     positiveTTL,
		negativeTTL:     negativeTTL,
		invalidWindow:   invalidWindow,
		invalidLimit:    invalidLimit,
		blockTTL:        blockTTL,
		touchInterval:   touchInterval,
	}
}

func (cache *APIKeyAuthCache) Lookup(scope string, digest string, now time.Time) (any, apiKeyCacheState) {
	if cache == nil || digest == "" {
		return nil, apiKeyCacheMiss
	}
	key := scopedAPIKey(scope, digest)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if entry, ok := cache.positive.get(key, now); ok {
		return entry.value, apiKeyCachePositive
	}
	if _, ok := cache.negative.get(key, now); ok {
		return nil, apiKeyCacheNegative
	}
	return nil, apiKeyCacheMiss
}

func (cache *APIKeyAuthCache) StorePositive(scope string, digest string, identity int64, value any, now time.Time) {
	if cache == nil || digest == "" {
		return
	}
	key := scopedAPIKey(scope, digest)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.negative.delete(key)
	cache.positive.set(apiKeyCacheEntry{
		key:       key,
		identity:  identity,
		value:     value,
		expiresAt: now.Add(cache.positiveTTL),
	})
}

func (cache *APIKeyAuthCache) StoreNegative(scope string, digest string, identity int64, now time.Time) {
	if cache == nil || digest == "" {
		return
	}
	key := scopedAPIKey(scope, digest)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.positive.delete(key)
	cache.negative.set(apiKeyCacheEntry{
		key:       key,
		identity:  identity,
		expiresAt: now.Add(cache.negativeTTL),
	})
}

func (cache *APIKeyAuthCache) InvalidateIdentity(scope string, identity int64) {
	if cache == nil || identity <= 0 {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.positive.deleteIdentity(scope, identity)
	cache.negative.deleteIdentity(scope, identity)
	cache.deleteTouchLocked(scopedAPIKey(scope, identityKey(identity)))
}

func (cache *APIKeyAuthCache) IsBlocked(clientID string, now time.Time) bool {
	if cache == nil || clientID == "" {
		return false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	element, ok := cache.limiters[clientID]
	if !ok {
		return false
	}
	entry := element.Value.(apiKeyLimiterEntry)
	if !entry.blockedUntil.IsZero() && now.Before(entry.blockedUntil) {
		cache.limiterOrder.MoveToFront(element)
		return true
	}
	if now.Sub(entry.windowStart) >= cache.invalidWindow {
		cache.removeLimiterElementLocked(element)
		return false
	}
	return false
}

func (cache *APIKeyAuthCache) RecordInvalid(clientID string, now time.Time) bool {
	if cache == nil || clientID == "" {
		return false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	entry := apiKeyLimiterEntry{clientID: clientID, windowStart: now}
	if element, ok := cache.limiters[clientID]; ok {
		entry = element.Value.(apiKeyLimiterEntry)
		if !entry.blockedUntil.IsZero() && now.Before(entry.blockedUntil) {
			cache.limiterOrder.MoveToFront(element)
			return true
		}
		if now.Sub(entry.windowStart) >= cache.invalidWindow {
			entry.windowStart = now
			entry.count = 0
			entry.blockedUntil = time.Time{}
		}
		entry.count++
		if entry.count >= cache.invalidLimit {
			entry.blockedUntil = now.Add(cache.blockTTL)
		}
		element.Value = entry
		cache.limiterOrder.MoveToFront(element)
		return !entry.blockedUntil.IsZero()
	}
	entry.count = 1
	if entry.count >= cache.invalidLimit {
		entry.blockedUntil = now.Add(cache.blockTTL)
	}
	element := cache.limiterOrder.PushFront(entry)
	cache.limiters[clientID] = element
	for cache.limiterOrder.Len() > cache.limiterCapacity {
		cache.removeLimiterElementLocked(cache.limiterOrder.Back())
	}
	return !entry.blockedUntil.IsZero()
}

func (cache *APIKeyAuthCache) ResetInvalid(clientID string) {
	if cache == nil || clientID == "" {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if element, ok := cache.limiters[clientID]; ok {
		cache.removeLimiterElementLocked(element)
	}
}

func (cache *APIKeyAuthCache) ReserveTouch(scope string, identity int64, now time.Time) bool {
	if cache == nil || identity <= 0 {
		return true
	}
	key := scopedAPIKey(scope, identityKey(identity))
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if element, ok := cache.touches[key]; ok {
		entry := element.Value.(apiKeyTouchEntry)
		if now.Sub(entry.reservedAt) < cache.touchInterval {
			cache.touchOrder.MoveToFront(element)
			return false
		}
		entry.reservedAt = now
		element.Value = entry
		cache.touchOrder.MoveToFront(element)
		return true
	}
	element := cache.touchOrder.PushFront(apiKeyTouchEntry{key: key, reservedAt: now})
	cache.touches[key] = element
	for cache.touchOrder.Len() > cache.touchCapacity {
		cache.removeTouchElementLocked(cache.touchOrder.Back())
	}
	return true
}

func (cache *APIKeyAuthCache) ReleaseTouch(scope string, identity int64, reservedAt time.Time) {
	if cache == nil || identity <= 0 {
		return
	}
	key := scopedAPIKey(scope, identityKey(identity))
	cache.mu.Lock()
	defer cache.mu.Unlock()
	element, ok := cache.touches[key]
	if !ok {
		return
	}
	entry := element.Value.(apiKeyTouchEntry)
	if entry.reservedAt.Equal(reservedAt) {
		cache.removeTouchElementLocked(element)
	}
}

func (cache *APIKeyAuthCache) removeLimiterElementLocked(element *list.Element) {
	if element == nil {
		return
	}
	entry := element.Value.(apiKeyLimiterEntry)
	delete(cache.limiters, entry.clientID)
	cache.limiterOrder.Remove(element)
}

func (cache *APIKeyAuthCache) deleteTouchLocked(key string) {
	if element, ok := cache.touches[key]; ok {
		cache.removeTouchElementLocked(element)
	}
}

func (cache *APIKeyAuthCache) removeTouchElementLocked(element *list.Element) {
	if element == nil {
		return
	}
	entry := element.Value.(apiKeyTouchEntry)
	delete(cache.touches, entry.key)
	cache.touchOrder.Remove(element)
}

func scopedAPIKey(scope string, key string) string {
	return scope + "\x00" + key
}

func cacheKeyScope(key string) string {
	for index := range key {
		if key[index] == 0 {
			return key[:index]
		}
	}
	return ""
}

func identityKey(identity int64) string {
	return strconv.FormatInt(identity, 10)
}
