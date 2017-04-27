package discovery

import (
  "fmt"
  "net"
  "perc/discovery/provider"
)

/**
 * Lookup a zone service
 */
func LookupTXT(d string, z provider.Zone) (string, error) {
  q := z.String()
  if d != "" {
    q += "."+ d
  }
  
  r, err := net.LookupTXT(q)
  if err != nil {
    return "", err
  }
  if len(r) < 1 {
    return "", fmt.Errorf("No records for zone: %v", q)
  }
  
  return r[0], nil
}