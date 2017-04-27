package etcd

import (
  "fmt"
  "net"
  "time"
  "strings"
  "perc/discovery"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
  "github.com/coreos/etcd/clientv3"
)

/**
 * Etcd-backed discovery service
 */
type Service struct {
  zones   []discovery.Zone
  clients map[string]*clientv3.Client
}

/**
 * Create a new discovery service
 */
func New(d string, z []discovery.Zone) (*Service, error) {
  clients := make(map[string]*clientv3.Client)
  
  for _, e := range z {
    c, err := clientForZone(d, e)
    if err != nil {
      alt.Errorf("etcd: Could not lookup discovery service: %v", err)
    }
    clients[e.String()] = c
    if debug.VERBOSE {
      alt.Debugf("etcd: Created etcd discovery client: %v -> %v", e, c)
    }
  }
  if len(clients) < 1 {
    return nil, fmt.Errorf("No discovery services available")
  }
  
  return &Service{z, clients}, nil
}

/**
 * Create a client for the specified zone
 */
func clientForZone(d string, z discovery.Zone) (*clientv3.Client, error) {
  
  q := z.String()
  if d != "" {
    q += "."+ d
  }
  
  r, err := net.LookupTXT(q)
  if err != nil {
    return nil, err
  }
  if len(r) < 1 {
    return nil, fmt.Errorf("No records for zone: %v", q)
  }
  
  nodes := strings.Split(r[0], ",")
  if debug.VERBOSE {
    alt.Debugf("etcd: Resolved discovery service: %v -> %v", q, strings.Join(nodes, ", "))
  }
  
  c, err := clientv3.New(clientv3.Config{Endpoints:nodes, DialTimeout:time.Second * 5})
  if err != nil {
    return nil, err
  }
  
  return c, nil
}

/**
 * Shutdown the service
 */
func (s *Service) Close() {
  for _, v := range s.clients {
    v.Close()
  }
}
