package etcd

import (
  "fmt"
  "time"
  "path"
  "strings"
  "context"
  "perc/discovery/provider"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
  "github.com/rcrowley/go-metrics"
  "github.com/coreos/etcd/clientv3"
)

const (
  keyPrefix = "/disc/perc"
)

const (
  timeout   = time.Second * 10
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
  zones   []provider.Zone
  clients []*clientv3.Client
}

/**
 * Create a new discovery service
 */
func New(d string, z []provider.Zone) (*Service, error) {
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
func clientForZone(d string, z provider.Zone) (*clientv3.Client, error) {
  
  r, err := provider.LookupTXT(d, z)
  if err != nil {
    return nil, err
  }
  
  nodes := strings.Split(r, ",")
  if debug.VERBOSE {
    alt.Debugf("etcd: Resolved discovery service: %v -> %v", z, strings.Join(nodes, ", "))
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
 * Register services
 */
func (s *Service) RegisterProviders(inst string, svcs map[string]string) (*provider.Lease, error) {
  if len(s.clients) < 1 {
    return nil, provider.ErrNoDiscovery
  }
  
  expires := time.Now().Add(timeout)
  for _, e := range s.clients {
    for k, v := range svcs {
      
      cxt, cancel := context.WithTimeout(context.Background(), timeout)
      grant, err := e.Grant(cxt, int64(timeout / time.Second))
      cancel()
      if err != nil {
        return nil, err
      }
      
      cxt, cancel = context.WithTimeout(context.Background(), timeout)
      _, err = e.Put(cxt, keyPath(keyPrefix, k, inst), v, clientv3.WithLease(grant.ID))
      cancel()
      if err != nil {
        return nil, err
      }
      
    }
  }
  
  return &provider.Lease{inst, svcs, expires}, nil
}

/**
 * Obtain the next service provider
 */
func (s *Service) LookupProvider(svc string) (string, error) {
  p, err := s.LookupProviders(1, svc)
  if err != nil {
    return "", err
  }
  if len(p) < 1 {
    return "", provider.ErrNoProviders
  }
  return p[0], nil
}

/**
 * Lookup a service
 */
func (s *Service) LookupProviders(n int, svc string) ([]string, error) {
  etcdLookupRate.Mark(1)
  start := time.Now()
  defer func(){
    etcdLookupDuration.Update(time.Since(start))
  }()
  
  var r []string
  if len(s.clients) < 1 {
    etcdLookupErrorRate.Mark(1)
    return nil, provider.ErrNoDiscovery
  }
  
  outer:
  for _, c := range s.clients {
    cxt, cancel := context.WithTimeout(context.Background(), timeout)
    rsp, err := c.Get(cxt, keyPath(keyPrefix, svc), clientv3.WithFromKey())
    cancel()
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
    return nil, provider.ErrNoProviders
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
