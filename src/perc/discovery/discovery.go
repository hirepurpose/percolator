package discovery

import (
  "fmt"
  "perc/discovery/etcd"
  "perc/discovery/provider"
)

/**
 * A discovery service
 */
type Service interface {
  ServiceProvider(string)(string, error)
  ServiceProviders(int, string)([]string, error)
}

/**
 * Create a discovery service
 */
func New(d, s string) (Service, error) {
  
  spec, err := provider.Parse(s)
  if err != nil {
    return nil, err
  }
  
  switch spec.Type {
    case "etcd":
      return etcd.New(d, spec.Zones)
  }
  
  return nil, fmt.Errorf("Unsupported discovery provider type: %v", spec.Type)
}
