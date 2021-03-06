package provider

import (
  "fmt"
  "time"
  "strings"
)

var (
  ErrMalformed    = fmt.Errorf("Malformed provider")
  ErrNoDiscovery  = fmt.Errorf("No discovery service available")
  ErrNoProviders  = fmt.Errorf("No providers available")
)

/**
 * Availability zone
 */
type Zone []string

// Parse a zone
func parseZone(s string) (Zone, error) {
  z := strings.Split(strings.TrimSpace(s), ".")
  if len(z) < 1 {
    return nil, ErrMalformed
  }
  for i, e := range z {
    z[i] = strings.TrimSpace(e)
  }
  return z, nil
}

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
 * Lookup the zone's hosts for the provided domain
 */
func (z Zone) Hosts(d string) ([]string, error) {
  
  r, err := LookupTXT(d, z)
  if err != nil {
    return nil, err
  }
  
  h := strings.Split(r, ",")
  for i, e := range h {
    h[i] = strings.TrimSpace(e)
  }
  
  return h, nil
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
func Parse(s string) (*Provider, error) {
  sep := "://"
  
  x := strings.Index(s, sep)
  if x < 1 {
    return nil, ErrMalformed
  }
  
  scheme := s[:x]
  s = s[x+len(sep):]
  
  var zones []Zone
  p := strings.Split(s, ",")
  for _, e := range p {
    z, err := parseZone(e)
    if err != nil {
      return nil, err
    }
    zones = append(zones, z)
  }
  
  if len(zones) < 1 {
    return nil, ErrMalformed
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

/**
 * A service registration lease
 */
type Lease struct {
  Instance  string
  Services  map[string]string
  Expires   time.Time
}
