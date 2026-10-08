package ai

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	// queryEmbedCacheSize caps entries so a burst of distinct queries can't grow the cache unbounded.
	queryEmbedCacheSize = 500
	// queryEmbedCacheTTL bounds how long a cached query vector is trusted; short enough that a mid-window
	// provider/model swap (caught separately by UpdateProviderConfig clearing the cache) isn't load-bearing.
	queryEmbedCacheTTL = 10 * time.Minute
)

type embedCacheEntry struct {
	key       string
	vec       []float32
	expiresAt time.Time
}

// queryEmbedCache is a small LRU+TTL cache for single-text query embeddings. RAG search and tag
// shortlisting re-embed the same or overlapping conversation text on every call (every tool
// invocation, every tag suggestion); caching those skips a provider round-trip for repeats within
// the same conversation turn. It deliberately does not cache GetEmbeddingsBatch: that path embeds
// knowledge-base content during (re)indexing, where inputs are already deduplicated by fingerprinting.
type queryEmbedCache struct {
	mu    sync.Mutex
	ll    *list.List
	items map[string]*list.Element
	size  int
	ttl   time.Duration
}

func newQueryEmbedCache(size int, ttl time.Duration) *queryEmbedCache {
	return &queryEmbedCache{
		ll:    list.New(),
		items: make(map[string]*list.Element),
		size:  size,
		ttl:   ttl,
	}
}

// get returns a copy of the cached vector so callers can never mutate shared cache state.
func (c *queryEmbedCache) get(key string) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*embedCacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.ll.Remove(el)
		delete(c.items, key)
		return nil, false
	}
	c.ll.MoveToFront(el)
	out := make([]float32, len(entry.vec))
	copy(out, entry.vec)
	return out, true
}

func (c *queryEmbedCache) put(key string, vec []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := make([]float32, len(vec))
	copy(stored, vec)

	if el, ok := c.items[key]; ok {
		entry := el.Value.(*embedCacheEntry)
		entry.vec = stored
		entry.expiresAt = time.Now().Add(c.ttl)
		c.ll.MoveToFront(el)
		return
	}
	el := c.ll.PushFront(&embedCacheEntry{key: key, vec: stored, expiresAt: time.Now().Add(c.ttl)})
	c.items[key] = el
	if c.ll.Len() > c.size {
		back := c.ll.Back()
		if back != nil {
			c.ll.Remove(back)
			delete(c.items, back.Value.(*embedCacheEntry).key)
		}
	}
}

// clear drops every cached vector; called whenever the embedding provider config changes so a
// stale vector from a previous model can never be served under a reused key.
func (c *queryEmbedCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ll.Init()
	c.items = make(map[string]*list.Element)
}

// embedCacheKey scopes the cache key to the provider endpoint, model and dimensions so a config
// change can only ever miss, never serve a vector from a different embedding space.
func embedCacheKey(baseURL, model string, dimensions int, text string) string {
	h := sha256.Sum256(fmt.Appendf(nil, "%s\x00%s\x00%d\x00%s", baseURL, model, dimensions, text))
	return hex.EncodeToString(h[:])
}
