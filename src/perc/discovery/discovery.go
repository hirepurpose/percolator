package discovery

import (
  "fmt"
  "strings"
)

var ErrMalformedProvider = fmt.Errorf("Malformed provider")

/**
 * A discovery service
 */
type Service interface {
  // AddressForService(int, string)([]string, error)
}

/**
 * Availability zone
 */
type Zone []string

/**
 * Display name
 */
func (z Zone) String() string {
  return strings.Join([]string(z), ".")
}

/**
 * Obtain the zone's region
 */
func (z Zone) Region() string {
  if l := len(z); l > 0 {
    return z[l-1]
  }else{
    return ""
  }
}

/**
 * Obtain the zone's availability zone
 */
func (z Zone) Zone() string {
  if l := len(z); l > 1 {
    return z[l-2]
  }else{
    return ""
  }
}

/**
 * Obtain the zone's rack
 */
func (z Zone) Rack() string {
  if l := len(z); l > 2 {
    return z[l-3]
  }else{
    return ""
  }
}

/**
 * Defines a discovery provider
 */
type Provider struct {
  Type  string
  Zones []Zone
}

/**
 * Parse a provider definition
 */
func ParseProvider(s string) (*Provider, error) {
  sep := "://"
  
  x := strings.Index(s, sep)
  if x < 1 {
    return nil, ErrMalformedProvider
  }
  
  scheme := s[:x]
  s = s[x+len(sep):]
  
  var zones []Zone
  p := strings.Split(s, ",")
  for _, e := range p {
    z := strings.Split(strings.TrimSpace(e), ".")
    if len(z) < 1 {
      return nil, ErrMalformedProvider
    }
    zones = append(zones, z)
  }
  
  if len(zones) < 1 {
    return nil, ErrMalformedProvider
  }
  
  return &Provider{scheme, zones}, nil
}

/**
 * Stringer
 */
func (p Provider) String() string {
  var s string
  for i, e := range p.Zones {
    if i > 0 { s += "," }
    s += e.String()
  }
  return p.Type +"://"+ s
}
