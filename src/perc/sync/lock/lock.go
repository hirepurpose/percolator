package lock

/**
 * A mutex
 */
type Mutex interface {
  Lock()(error)
  Unlock()(error)
  Perform(func()error)(error)
}
