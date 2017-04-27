package etcd

import (
  "fmt"
  "net"
  "time"
  "path"
  "strings"
  "context"
  "perc/discovery"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
  "github.com/coreos/etcd/clientv3"
)

const (
  discPrefix = "/disc/perc"
)

const (
  timeout = time.Second * 5
)

/**
 * Etcd-backed discovery service
 */
type Service struct {
  zones   []discovery.Zone
  clients []*clientv3.Client
}

/**
 * Create a new discovery service
 */
func New(d string, z []discovery.Zone) (*Service, error) {
  clients := make([]*clientv3.Client, 0)
  
  for _, e := range z {
    c, err := clientForZone(d, e)
    if err != nil {
      alt.Errorf("etcd: Could not lookup discovery service: %v", err)
    }
    clients = append(clients, c)
    if debug.VERBOSE {
      alt.Debugf("etcd: Created etcd discovery client for zone: %v", e)
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
 * Build a path for the provided keys
 */
func keyPath(k ...string) string {
  var p string
  for _, e := range k {
    if p != "" {
      p = path.Join(append([]string{p}, strings.Split(e, ".")...)...)
    }else{
      p = path.Join(strings.Split(e, ".")...)
    }
  }
  return p
}

/**
 * Lookup a service
 */
func (s *Service) AddressForService(n int, svc string) ([]string, error) {
  for _, c := range s.clients {
    
    cxt, cancel := context.WithTimeout(context.Background(), timeout)
    rsp, err := c.Get(cxt, keyPath(discPrefix, svc))
    defer cancel()
    if err != nil {
      return nil, err
    }
    
    for _, e := range rsp.Kvs {
      fmt.Println("YO", string(e.Value))
    }
    
  }
  return nil, discovery.ErrNoDiscovery
}

/**
 * Shutdown the service
 */
func (s *Service) Close() {
  for _, v := range s.clients {
    v.Close()
  }
}
