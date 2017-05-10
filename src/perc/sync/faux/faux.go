package faux

import (
  "perc/sync/lock"
)

/**
 * A faux mutex
 */
type Mutex struct {}

func (m Mutex) Lock() error {
  return nil
}

func (m Mutex) Unlock() error {
  return nil
}

func (m Mutex) Perform(f func()error) error {
  return f()
}

/**
 * A faux lock service
 */
type Service struct {}

func New() Service {
  return Service{}
}

func (s Service) Mutex(string) (lock.Mutex, error) {
  return Mutex{}, nil
}
