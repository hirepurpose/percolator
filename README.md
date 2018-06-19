# Percolator

Percolator is a service which provides **service discovery** and **distributed locking** for cloud-based systems, and **routes connections** between internal service components. It provides for progressive fail-over to other availability zones or datacenters in the event of service unavailability.

Percolator uses a backend service for coordination. Currently [Etcd](https://github.com/coreos/etcd/) is the only such supported service.

Better documentation is forthcoming.