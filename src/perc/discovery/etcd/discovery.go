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
  "github.com/rcrowley/go-metrics"
  "github.com/coreos/etcd/clientv3"
)

const (
  discPrefix = "/disc/perc"
)

const (
  timeout = time.Second * 5
)

var (
  etcdLookupRate metrics.Meter
  etcdLookupErrorRate metrics.Meter
  etcdLookupDuration metrics.Timer
)

func init() {
  etcdLookupRate = metrics.NewMeter()
  metrics.Register("percolator.etcd.lookup.rate", etcdLookupRate)
  etcdLookupErrorRate = metrics.NewMeter()
  metrics.Register("percolator.etcd.lookup.error.rate", etcdLookupErrorRate)
  etcdLookupDuration = metrics.NewTimer()
  metrics.Register("percolator.etcd.lookup.duration", etcdLookupDuration)
}

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
 * Obtain the next service provider
 */
func (s *Service) ServiceProvider(svc string) (string, error) {
  p, err := s.ServiceProviders(1, svc)
  if err != nil {
    return "", err
  }
  if len(p) < 1 {
    return "", discovery.ErrNoProviders
  }
  return p[0], nil
}

/**
 * Lookup a service
 */
func (s *Service) ServiceProviders(n int, svc string) ([]string, error) {
  etcdLookupRate.Mark(1)
  start := time.Now()
  defer func(){
    etcdLookupDuration.Update(time.Since(start))
  }()
  
  var r []string
  if len(s.clients) < 1 {
    etcdLookupErrorRate.Mark(1)
    return nil, discovery.ErrNoDiscovery
  }
  
  outer:
  for _, c := range s.clients {
    cxt, cancel := context.WithTimeout(context.Background(), timeout)
    rsp, err := c.Get(cxt, keyPath(discPrefix, svc), clientv3.WithFromKey())
    defer cancel()
    if err != nil {
      etcdLookupErrorRate.Mark(1)
      return nil, err
    }
    for _, e := range rsp.Kvs {
      r = append(r, string(e.Value))
      if len(r) > n {
        break outer
      }
    }
  }
  
  if len(r) < 1 {
    etcdLookupErrorRate.Mark(1)
    return nil, discovery.ErrNoProviders
  }
  return r, nil
}

/**
 * Shutdown the service
 */
func (s *Service) Close() {
  for _, v := range s.clients {
    v.Close()
  }
}
