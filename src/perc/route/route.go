package route

import (
  "fmt"
  "sync"
  "strings"
)

/**
 * A route maps a port to a backend
 */
type Route struct {
  sync.Mutex
  Listen    string
  Backends  []string
  Service   bool
  index     int
}

/**
 * Parse a route
 */
func Parse(s string) (*Route, error) {
  
  p := strings.Split(s, "=")
  if len(p) != 2 {
    return nil, fmt.Errorf("Invalid route; expected <listen>=<backend>[,...,<backendN>] in: %v", s)
  }
  
  listen := p[0]
  var backends []string
  var service bool
  
  for _, e := range strings.Split(p[1], ",") {
    e = strings.TrimSpace(e)
    backends = append(backends, e)
    if strings.Index(e, ":") < 0 {
      service = true
    }else if service {
      return nil, fmt.Errorf("Cannot mix service and host backend in a single route: %v", s)
    }
  }
  
  if len(backends) < 1 {
    return nil, fmt.Errorf("No backends defined in route: %v", s)
  }
  if service && len(backends) > 1 {
    return nil, fmt.Errorf("Only one service backend may be defined in a single route: %v", s)
  }
  
  return &Route{sync.Mutex{}, listen, backends, service, 0}, nil
}

/**
 * Obtain the next backend in the rotation
 */
func (r *Route) NextBackend() string {
  if len(r.Backends) == 1 {
    return r.Backends[0]
  }
  r.index++
  return r.Backends[r.index % len(r.Backends)]
}

/**
 * Stringer
 */
func (r Route) String() string {
  return r.Listen +" -> "+ strings.Join(r.Backends, ", ")
}
