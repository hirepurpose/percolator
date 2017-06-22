package service

import (
  "sync"
)

// Counter map key/val
type entry struct {
  Key     string
  Val     int64
  Client  string
}

// Counter map
type cmap struct {
  sync.RWMutex
  m map[string]int64
  u chan entry
}

// Create a counter map
func newCmap() *cmap {
  return &cmap{sync.RWMutex{}, make(map[string]int64), nil}
}

// Obtain a copy of the underlying map
func (c *cmap) Copy() map[string]int64 {
  c.RLock()
  defer c.RUnlock()
  d := make(map[string]int64)
  for k, v := range c.m {
    d[k] = v
  }
  return d
}

// Obtain the update channel
func (c *cmap) Put() chan<- entry {
  c.Lock()
  defer c.Unlock()
  if c.u == nil {
    c.u = make(chan entry, 64)
    c.update()
  }
  return c.u
}

// Run the update routine
func (c *cmap) update() {
  go func(){
    for e := range c.u {
      c.Lock()
      if b, ok := c.m[e.Key]; ok {
        c.m[e.Key] = b + e.Val
      }else{
        c.m[e.Key] = e.Val
      }
      c.Unlock()
    }
  }()
}
