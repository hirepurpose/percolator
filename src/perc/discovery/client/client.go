package client

import (
  "time"
  "perc/discovery"
)

import (
  "github.com/bww/go-alert"
)

/**
 * A client
 */
type Client struct {
  service discovery.Service
}

/**
 * Create a client
 */
func New(s discoveryService) *Client {
  return &Client{s}
}

/**
 * Register a single service and repeatedly renew our lease forever
 */
func (c *Client) Register(inst, svc, addr string) {
  go c.register(inst, svc, addr)
}

/**
 * Actually do it
 */
func (c *Client) register(inst string, svcs map[string]string) {
  for {
    wait := time.Second * 5 // default wait
    l, err := c.service.RegisterProvider(inst, svcs)
    if err != nil {
      alt.Errorf("discovery: Could not register local services: %v", err)
    }else{
      wait = time.Now().Sub(l.Expires) / 2 // wait half the duration until expiration
    }
    <- time.After(wait)
  }
}
