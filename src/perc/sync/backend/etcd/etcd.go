package etcd

import (
  "time"
  "path"
  "context"
  "perc/sync/lock"
  "perc/discovery/provider"
  "perc/discovery/backend/etcd"
)

import (
  "github.com/coreos/etcd/clientv3"
  "github.com/coreos/etcd/clientv3/concurrency"
)

const (
  keyPrefix = "/sync/perc"
)

const (
  timeout   = time.Second * 30
)

/**
 * Etcd-backed sync service
 */
type Service struct {
  zone    provider.Zone
  client  *clientv3.Client
}

/**
 * Create a new sync service
 */
func New(d string, z provider.Zone) (*Service, error) {
  c, err := etcd.ClientForZone(d, z)
  if err != nil {
    return nil, err
  }
  return &Service{z, c}, nil
}

/**
 * Register services
 */
func (s *Service) Mutex(key string) (lock.Mutex, error) {
  sess, err := concurrency.NewSession(s.client)
  if err != nil {
    return nil, err
  }
  return &mutex{concurrency.NewMutex(sess, path.Join(keyPrefix, key))}, nil
}

/**
 * Shutdown the service
 */
func (s *Service) Close() {
  s.client.Close()
}

/**
 * A mutex that conforms to perc/sync.Mutex
 */
type mutex struct {
  *concurrency.Mutex
}

/**
 * Lock it
 */
func (m mutex) Lock() error {
  cxt, cancel := context.WithTimeout(context.Background(), timeout)
  err := m.Mutex.Lock(cxt)
  cancel()
  return err
}

/**
 * Unlock it
 */
func (m mutex) Unlock() error {
  cxt, cancel := context.WithTimeout(context.Background(), timeout)
  err := m.Mutex.Unlock(cxt)
  cancel()
  return err
}

/**
 * Lock, exec, unlock it
 */
func (m mutex) Perform(f func()error) error {
  lerr := m.Lock()
  if lerr != nil {
    return lerr
  }
  ferr := f()
  lerr  = m.Unlock()
  return coalesce(ferr, lerr)
}

/**
 * Obtain the first non-nil error or nil if all errors are nil
 */
func coalesce(errs ...error) error {
  for _, e := range errs {
    if e != nil {
      return e
    }
  }
  return nil
}
