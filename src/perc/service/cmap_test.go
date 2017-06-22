package service

import (
  "fmt"
  "time"
  "testing"
)

import (
  "github.com/stretchr/testify/assert"
)

func TestCmap(t *testing.T) {
  m := newCmap()
  c := m.Put()
  n := int64(100000)
  for i := int64(0); i < n; i++ {
    c <- entry{string('A'+ rune(i % 10)), 1, string('L'+ rune(i % 10))}
  }
  
  close(c)
  <- time.After(time.Second)
  
  d := m.Copy()
  fmt.Println(d)
  
  var x int64
  for _, v := range d {
    x += v
  }
  
  assert.Equal(t, x, n)
}

