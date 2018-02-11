package route

import (
  "fmt"
  "sync"
  "strings"
  "unicode"
)

import (
  "github.com/bww/go-util/scan"
)

const (
  paramDelimOpen    = '('
  paramDelimClose   = ')'
  paramDelimAssign  = '='
  paramDelimList    = ','
  paramDelimQuote   = '\''
  paramDelimEsc     = '\\'
)

// A route maps a port to a backend
type Route struct {
  sync.Mutex
  Listen    string
  Backends  []string
  Service   bool
  index     int
}

// Parse a route
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

// Obtain the next backend in the rotation
func (r *Route) NextBackend() string {
  if len(r.Backends) == 1 {
    return r.Backends[0]
  }
  r.index++
  return r.Backends[r.index % len(r.Backends)]
}

// Stringer
func (r Route) String() string {
  return r.Listen +" -> "+ strings.Join(r.Backends, ", ")
}

// A backend configuration
type Backend struct {
  Name    string
  Params  map[string]string
}

// Parse a backend in the form: host|service[{key1=value1[,...]}]
func parseBackend(s string) (*Backend, error) {
  var err error
  
  n := strings.IndexFunc(s, func(r rune) bool {
    return unicode.IsSpace(r) || r == paramDelimOpen
  })
  if n < 0 {
    return &Backend{Name:s}, nil
  }
  
  name := s[:n]
  _, s = scan.White(s[n:])
  
  var params map[string]string
  if len(s) > 0 && s[0] == paramDelimOpen {
    params, err = parseParams(s)
    if err != nil {
      return nil, err
    }
  }
  
  return &Backend{name, params}, nil
}

// Parse parameters in the form: (key1=value1[,...])
func parseParams(s string) (map[string]string, error) {
  if len(s) < 1 || s[0] != paramDelimOpen {
    return nil, fmt.Errorf("Invalid parameters; expected '%v', got '%v'", string(paramDelimOpen), string(s[0]))
  }else{
    s = s[1:]
  }
  
  params := make(map[string]string)
  for len(s) > 0 {
    _, s = scan.White(s)
    
    if len(s) < 1 {
      return nil, fmt.Errorf("Unexpected end of parameters")
    }
    if s[0] == paramDelimClose {
      break
    }
    if s[0] == paramDelimList {
      s = s[1:]
      continue
    }
    
    var k, v string
    var err error
    k, v, s, err = parseKeyValue(s)
    if err != nil {
      return nil, err
    }
    
    params[k] = v
  }
  
  return params, nil
}

// Parse a key/value pair in the form: (key1='value1')
func parseKeyValue(s string) (string, string, string, error) {
  var key, val string
  var err error
  
  key, s, err = scan.Ident(s)
  if err != nil {
    return "", "", "", err
  }
  
  _, s = scan.White(s)
  if len(s) < 1 || s[0] != paramDelimAssign {
    return "", "", "", fmt.Errorf("Expected '=', got '%v'", string(s[0]))
  }else{
    s = s[1:]
  }
  
  _, s = scan.White(s)
  val, s, err = scan.String(s, paramDelimQuote, paramDelimEsc)
  if err != nil {
    return "", "", "", err
  }
  
  return key, val, s, nil
}
