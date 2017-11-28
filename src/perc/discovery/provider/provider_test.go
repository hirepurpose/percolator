package provider

import (
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestZones(t *testing.T) {
  var err error
  var z Zone
  
  z, err = parseZone("us-east-1")
  if assert.Nil(t, err) {
    assert.Equal(t, "us-east-1", z.Region())
    assert.Equal(t, "", z.Zone())
    assert.Equal(t, "", z.Rack())
  }
  
  z, err = parseZone("zone.us-east-1")
  if assert.Nil(t, err) {
    assert.Equal(t, "us-east-1", z.Region())
    assert.Equal(t, "zone", z.Zone())
    assert.Equal(t, "", z.Rack())
  }
  
  z, err = parseZone("rack.zone.us-east-1")
  if assert.Nil(t, err) {
    assert.Equal(t, "us-east-1", z.Region())
    assert.Equal(t, "zone", z.Zone())
    assert.Equal(t, "rack", z.Rack())
  }
  
  z, err = parseZone("too-much-detail.rack.zone.us-east-1")
  if assert.Nil(t, err) {
    assert.Equal(t, "us-east-1", z.Region())
    assert.Equal(t, "zone", z.Zone())
    assert.Equal(t, "rack", z.Rack())
  }
  
  z, err = parseZone("us-east-1")
  if assert.Nil(t, err) {
    h, err := z.Hosts("debug.disc.hirepurpose.com")
    if assert.Nil(t, err) {
      assert.Equal(t, []string{"localhost:2379"}, h)
    }
  }
  
}