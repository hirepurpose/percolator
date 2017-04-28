package registry

import (
  "net"
)

import (
  "github.com/bww/go-util/env"
)

/**
 * Attempt to determine the public address of the instance on which
 * this service is running.
 */
func PublicAddr() string {
  return env.LocalAddr() // use the internal address
}

/**
 * Rebase an address for a host
 */
func RebaseAddr(host, addr string) (string, error) {
  _, p, err := net.SplitHostPort(addr)
  if err != nil {
    return "", err
  }
  return host +":"+ p, nil
}
