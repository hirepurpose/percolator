package discovery

/**
 * A discovery service
 */
type Service interface {
  AddressForService(int, string)([]string, error)
}
