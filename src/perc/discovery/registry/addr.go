package registry

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
