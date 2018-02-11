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
  Backends  []Backend
  Service   bool
  index     int
}

// Parse a route
func Parse(s string) (*Route, error) {
  var err error
  
  n := strings.IndexRune(s, '=')
  if n < 0 {
    return nil, fmt.Errorf("Invalid route; expected <listen>=<backend>[,...,<backendN>] in: %v", s)
  }
  
  listen := strings.TrimSpace(s[:n])
  _, s = scan.White(s[n+1:])
  
  var service bool
  var backends []Backend
  for i := 0; len(s) > 0; i++ {
    var b Backend
    b, s, err = parseBackend(s)
    if err != nil {
      return nil, err
    }
    v := strings.IndexRune(b.Name, ':') < 0
    if i == 0 {
      service = v
    }else if service != v {
      return nil, fmt.Errorf("Cannot mix host and service backends in the same route")
    }
    backends = append(backends, b)
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
func parseBackend(s string) (Backend, string, error) {
  var err error
  
  n := strings.IndexFunc(s, func(r rune) bool {
    return unicode.IsSpace(r) || r == paramDelimOpen
  })
  if n < 0 {
    return Backend{Name:s}, "", nil
  }
  
  name := s[:n]
  _, s = scan.White(s[n:])
  
  var params map[string]string
  if len(s) > 0 && s[0] == paramDelimOpen {
    params, s, err = parseParams(s)
    if err != nil {
      return Backend{}, "", err
    }
  }
  
  return Backend{name, params}, s, nil
}

// Parse parameters in the form: (key1=value1[,...])
func parseParams(s string) (map[string]string, string, error) {
  if len(s) < 1 || s[0] != paramDelimOpen {
    return nil, "", fmt.Errorf("Invalid parameters; expected '%v', got '%v'", string(paramDelimOpen), string(s[0]))
  }else{
    s = s[1:]
  }
  
  params := make(map[string]string)
  for len(s) > 0 {
    _, s = scan.White(s)
    
    if len(s) < 1 {
      return nil, "", fmt.Errorf("Unexpected end of parameters")
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
      return nil, "", err
    }
    
    params[k] = v
  }
  
  return params, s, nil
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
