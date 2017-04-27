package etcd

import (
  "fmt"
  "net"
  "time"
  "path"
  "strings"
  "context"
  "perc/client"
  "perc/discovery"
  "perc/discovery/etcd"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
  "github.com/coreos/etcd/clientv3"
)

const (
  timeout = time.Second * 5
)

/**
 * Etcd-backed provider
 */
type Provider struct {
  zones   []discovery.Zone
  clients []*clientv3.Client
}

/**
 * Create a new discovery service
 */
func New(d string, z []discovery.Zone) (*Provider, error) {
  clients := make([]*clientv3.Client, 0)
  
  for _, e := range z {
    c, err := etcd.ClientForZone(d, e)
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
  
  return &Provider{z, clients}, nil
}

/**
 * Register services
 */
func (p *Provider) RegisterServices(inst string, svcs map[string]string) (*client.Lease, error) {
  if len(p.clients) < 1 {
    return nil, discovery.ErrNoDiscovery
  }
  
  for _, c := range p.clients {
    for k, v := range svcs {
      cxt, cancel := context.WithTimeout(context.Background(), timeout)
      rsp, err := c.Put(cxt, etcd.KeyPath(discPrefix, k, inst))
      defer cancel()
      if err != nil {
        return nil, err
      }
    }
  }
  
  return r, nil
}

/**
 * Shutdown the service
 */
func (s *Provider) Close() {
  for _, v := range p.clients {
    v.Close()
  }
}
