package service

import (
  "fmt"
  "testing"
  "math/rand"
)

import (
  // "github.com/stretchr/testify/assert"
)

func TestCmap(t *testing.T) {
  m := newCmap()
  c := m.Put()
  for i := 0; i < 100000; i++ {
    v := rand.Int63n(10)
    c <- keyval{string('A'+ rune(v)), v}
  }
  fmt.Println(m.Copy())
  // assert.Equal()
}

