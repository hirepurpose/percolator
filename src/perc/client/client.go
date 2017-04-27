package client

import (
  "fmt"
)

var (
  ErrNoDiscovery  = fmt.Errorf("No discovery service available")
)

/**
 * A service registration lease
 */
type Lease struct {
  Instance  string
  Services  map[string]string
  Expires   time.Time
}

/**
 * A service provider
 */
type Provider interface {
  RegisterServices(string, map[string]string)(*Lease, error)
}
