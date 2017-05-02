package discovery

import (
  "fmt"
  "perc/discovery/provider"
  "perc/discovery/backend/etcd"
)

/**
 * A discovery service
 */
type Service interface {
  RegisterProviders(string, map[string]string)(*provider.Lease, error)
  LookupProvider(string)(string, error)
  LookupProviders(int, string)([]string, error)
}

/**
 * Create a discovery service
 */
func New(d, s string) (Service, error) {
  
  spec, err := provider.Parse(s)
  if err != nil {
    return nil, err
  }
  if len(spec.Zones) < 1 {
    return nil, fmt.Errorf("No zones specified")
  }
  
  switch spec.Type {
    case "etcd":
      return etcd.New(d, spec.Zones)
  }
  
  return nil, fmt.Errorf("Unsupported discovery provider type: %v", spec.Type)
}
