package route

import (
  "fmt"
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestParseBackend(t *testing.T) {
  testParseBackend(t, `hello(a='Value')`, "hello", map[string]string{"a": "Value"}, nil)
  testParseBackend(t, `hello (a='Value')`, "hello", map[string]string{"a": "Value"}, nil)
  testParseBackend(t, `hello ( key_name = 'Value' )`, "hello", map[string]string{"key_name": "Value"}, nil)
  testParseBackend(t, `hello/123:456(key_name='Value\nhere')`, "hello/123:456", map[string]string{"key_name": "Value\nhere"}, nil)
  testParseBackend(t, `hello/123:456 (key_name='Value', another='Check it')`, "hello/123:456", map[string]string{"key_name": "Value", "another": "Check it"}, nil)
  testParseBackend(t, `hello/123:456( key_name='Value', another='Check it' )`, "hello/123:456", map[string]string{"key_name": "Value", "another": "Check it"}, nil)
  testParseBackend(t, `hello/123:456 ( key_name='Value', another='Check it' ) trailing stuff, which we ignore`, "hello/123:456", map[string]string{"key_name": "Value", "another": "Check it"}, nil)
}

func testParseBackend(t *testing.T, in, en string, ep map[string]string, eerr error) bool {
  b, _, aerr := parseBackend(in)
  if aerr != nil || eerr != nil {
    fmt.Printf("%v -> %v\n", in, aerr)
    return assert.Equal(t, eerr, aerr, "Errors do not match")
  }
  fmt.Printf("%v -> [%v] %v\n", in, b.Name, b.Params)
  res := true
  res = res && assert.Equal(t, en, b.Name, "Names do not match")
  res = res && assert.Equal(t, ep, b.Params, "Params do not match")
  return res
}

func TestParseRoute(t *testing.T) {
  testParseRoute(t, `:9000=upstream`, &Route{Listen:":9000", Backends:[]Backend{{Name:"upstream"}}, Service:true}, nil)
}

func testParseRoute(t *testing.T, in string, er *Route, eerr error) bool {
  ar, aerr := Parse(in)
  if aerr != nil || eerr != nil {
    fmt.Printf("%v -> %v\n", in, aerr)
    return assert.Equal(t, eerr, aerr, "Errors do not match")
  }
  res := true
  return assert.Equal(t, er, ar, "Routes do not match")
}
