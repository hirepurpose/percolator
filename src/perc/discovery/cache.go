package discovery

import (
  "time"
  "sync"
  "strings"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
)

const (
  DefaultTimeout    = time.Minute * 5
  DefaultMaxRecords = 100
)

/**
 * A cache entry
 */
type cacheEntry struct {
  sync.Mutex
  providers []string
  index     int
  expiry    time.Time
}

/**
 * Obtain the next provider
 */
func (e *cacheEntry) Next(n int) []string {
  e.Lock()
  defer e.Unlock()
  var r []string
  
  l := len(e.providers)
  b := e.index % l
  u := b + n
  a := n
  
  if u > l {
    a  = l - b
    x := n - a // spillover
    if x > b {
      x = b
    }
    a += x
    r = append(e.providers[b:u], e.providers[:x]...)
  }else{
    r = e.providers[b:u]
  }
  
  e.index += a
  return r
}

/**
 * A caching discovery service
 */
type Cache struct {
  sync.Mutex
  service     Service
  timeout     time.Duration
  cache       map[string]*cacheEntry
  maxRecords  int
}

/**
 * Create a caching service which wraps an underlying service
 */
func NewCache(s Service, t time.Duration) *Cache {
  return &Cache{sync.Mutex{}, s, t, make(map[string]*cacheEntry), DefaultMaxRecords}
}

/**
 * Obtain the next service provider
 */
func (c *Cache) ServiceProvider(svc string) (string, error) {
  r, err := c.ServiceProviders(1, svc)
  if err != nil {
    return "", err
  }
  if len(r) < 1 {
    return "", ErrNoProviders
  }
  return r[0], nil
}

/**
 * Lookup a service
 */
func (c *Cache) ServiceProviders(n int, svc string) ([]string, error) {
  c.Lock()
  defer c.Unlock()
  now := time.Now()
  
  e, ok := c.cache[svc]
  if !ok || now.After(e.expiry) {
    if debug.VERBOSE {
      alt.Debugf("cache: Querying for providers: %v", svc)
    }
    r, err := c.service.ServiceProviders(c.maxRecords, svc)
    if err != nil {
      return nil, err
    }
    e = &cacheEntry{sync.Mutex{}, r, 0, now.Add(c.timeout)}
    c.cache[svc] = e
    if debug.VERBOSE {
      alt.Debugf("cache: Received %d providers: %v -> %v", len(r), svc, strings.Join(r, ", "))
    }
  }
  
  return e.Next(n), nil
}
