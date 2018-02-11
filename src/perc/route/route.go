package route

import (
  "fmt"
  "sync"
  "sync/atomic"
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

// A syntax error
type syntaxError error

// Is it a syntax error?
func IsSyntaxError(e error) bool {
  _, ok := e.(syntaxError)
  return ok
}

// A route maps a port to a backend
type Route struct {
  sync.Mutex
  Listen    string
  Backends  []Backend
  Service   bool
  index     uint64
}

// Parse a route
func Parse(s string) (*Route, error) {
  var err error
  p := s
  
  n := strings.IndexRune(s, '=')
  if n < 0 {
    return nil, syntaxError(fmt.Errorf("Invalid route; expected <listen>=<backend>[,...,<backendN>] in: %v", p))
  }
  
  listen := strings.TrimSpace(s[:n])
  _, s = scan.White(s[n+1:])
  
  var service bool
  var backends []Backend
  for i := 0; len(s) > 0; i++ {
    var b Backend
    
    if i > 0 {
      if s[0] != ',' {
        return nil, syntaxError(fmt.Errorf("Missing ',' in backend list"))
      }else{
        _, s = scan.White(s[1:])
      }
    }
    
    b, s, err = parseBackend(s)
    if err != nil {
      return nil, err
    }
    if b.Name == "" {
      return nil, syntaxError(fmt.Errorf("Backend is empty"))
    }
    
    backends = append(backends, b)
    
    v := strings.IndexRune(b.Name, ':') < 0
    if i == 0 {
      service = v
    }else if service != v {
      return nil, syntaxError(fmt.Errorf("Cannot mix host and service backends in the same route"))
    }
    
    _, s = scan.White(s)
  }
  
  if len(backends) < 1 {
    return nil, syntaxError(fmt.Errorf("No backends defined in route: %v", p))
  }
  if service && len(backends) > 1 {
    return nil, fmt.Errorf("Only one service backend may be defined in a single route: %v", p)
  }
  
  return &Route{sync.Mutex{}, listen, backends, service, 0}, nil
}

// Obtain the next backend in the rotation
func (r *Route) NextBackend() Backend {
  if len(r.Backends) == 1 {
    return r.Backends[0]
  }
  n := atomic.AddUint64(&r.index, 1)
  return r.Backends[int(n) % len(r.Backends)]
}

// Stringer
func (r Route) String() string {
  var b string
  for i, e := range r.Backends {
    if i > 0 { b += ", " }
    b += e.String()
  }
  return r.Listen +" -> "+ b
}

// A backend configuration
type Backend struct {
  Name    string
  Params  map[string]string
}

// Stringer
func (b Backend) String() string {
  s := b.Name
  if len(b.Params) > 0 {
    s += "("
    for k, v := range b.Params {
      s += k +"='"+ scan.Escape(v, paramDelimQuote, paramDelimEsc) +"'"
    }
    s += ")"
  }
  return s
}

// Parse a backend in the form: host|service[{key1=value1[,...]}]
func parseBackend(s string) (Backend, string, error) {
  var err error
  
  n := strings.IndexFunc(s, func(r rune) bool {
    return unicode.IsSpace(r) || r == paramDelimOpen || r == paramDelimList
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
    return nil, "", syntaxError(fmt.Errorf("Invalid parameters; expected '%v', got '%v'", string(paramDelimOpen), string(s[0])))
  }else{
    s = s[1:]
  }
  
  params := make(map[string]string)
  for len(s) > 0 {
    _, s = scan.White(s)
    
    if len(s) < 1 {
      return nil, "", syntaxError(fmt.Errorf("Unexpected end of parameters"))
    }
    if s[0] == paramDelimClose {
      s = s[1:]
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
  if len(s) < 1 {
    return "", "", "", syntaxError(fmt.Errorf("Unexpected end of input"))
  }
  
  // flag style; key with no value
  if s[0] == paramDelimClose {
    return key, "", s, nil
  }else if s[0] == paramDelimList {
    return key, "", s, nil
  }
  
  // otherwise next value must 
  if s[0] != paramDelimAssign {
    return "", "", "", syntaxError(fmt.Errorf("Expected '=', got '%v'", string(s[0])))
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
