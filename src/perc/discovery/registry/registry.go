package registry

import (
  "time"
  "perc/discovery"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/rand"
  "github.com/bww/go-util/debug"
)

/**
 * A service registry
 */
type Registry struct {
  service discovery.Service
}

/**
 * Create a client
 */
func New(s discovery.Service) *Registry {
  return &Registry{s}
}

/**
 * Publish a single service and repeatedly renew our lease forever
 */
func (r *Registry) Publish(inst, svc, addr string) {
  go r.publish(inst, map[string]string{svc: addr})
}

/**
 * Publish
 */
func (r *Registry) publish(inst string, svcs map[string]string) {
  name := inst +"-"+ rand.RandomString(16)
  for {
    wait := time.Second * 10 // default wait
    if debug.TRACE {
      alt.Debugf("discovery: Publishing services: <%v> %v", name, svcs)
    }
    l, err := r.service.RegisterProviders(name, svcs)
    if err != nil {
      alt.Errorf("discovery: Could not register local services: %v", err)
    }else{
      wait = l.Expires.Sub(time.Now()) / 2 // wait half the duration until expiration
    }
    if wait < time.Second {
      wait = time.Second // sanity check
    }
    <- time.After(wait)
  }
}
