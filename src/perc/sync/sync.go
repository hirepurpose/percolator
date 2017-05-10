package sync

import (
  "fmt"
  "perc/sync/lock"
  "perc/sync/backend/etcd"
  "perc/discovery/provider"
)

/**
 * A sync service
 */
type Service interface {
  Mutex(string)(lock.Mutex, error)
}

/**
 * Create a sync service
 */
func New(d, s string) (Service, error) {
  
  spec, err := provider.Parse(s)
  if err != nil {
    return nil, err
  }
  if len(spec.Zones) != 1 {
    return nil, fmt.Errorf("Sync service must use exactly one zone to coordinate clients; got %v", len(spec.Zones))
  }
  
  switch spec.Type {
    case "etcd":
      return etcd.New(d, spec.Zones[0])
  }
  
  return nil, fmt.Errorf("Unsupported sync provider type: %v", spec.Type)
}
