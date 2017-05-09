package main

import (
  "os"
  "fmt"
  "flag"
  "time"
  "strings"
  "net/http"
  "crypto/sha1"
  "perc/route"
  "perc/service"
  "perc/discovery"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-alert/sentry"
  "github.com/bww/go-util/rand"
  "github.com/bww/go-util/debug"
  "github.com/bww/go-metrics-influxdb"
  "github.com/rcrowley/go-metrics"
)

/**
 * You know what it does
 */
func main() {
  var proxyRoutes flagList
  
  cmdline       := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
  fProfile      := cmdline.String   ("profile",       coalesce(os.Getenv("HP_API_PPROF"), "localhost:2221"),                "The interface and port to accept pprof (profiling) connections on.")
  fDomain       := cmdline.String   ("domain",        coalesce(os.Getenv("HP_DISCOVERY_DOMAIN"), "disc.hirepurpose.com"),   "The domain to use for service discovery.")
  fDiscovery    := cmdline.String   ("discovery",     coalesce(os.Getenv("HP_DISCOVERY_SERVICE"), "etcd://us-east-1"),      "The discovery service used for service lookup, specified as 'service://[az.]region[,..,[azN.]regionN]'. Regions should be provided in descending order of preference.")
  fInflux       := cmdline.String   ("influxdb",      os.Getenv("HP_METRICS_INFLUXDB"),                                     "The InfluxDB metrics reporting backend, specified as: 'host[:port]'.")
  fEnviron      := cmdline.String   ("environ",       coalesce(os.Getenv("HP_ENVIRON"), os.Getenv("ENVIRON"), "devel"),     "The environment in which the service is running (devel, staging, production).")
  fSentry       := cmdline.String   ("sentry",        os.Getenv("HP_SENTRY"),                                               "Report errors to Sentry. The Sentry authentication DSN should be provided as an argument.")
  fIOTimeout    := cmdline.Duration ("timeout",       strToDur(coalesce(os.Getenv("HP_TIMEOUT"), "0")),                     "Specify both the read and write timeouts for client connections at once. This flag overrides -timeout:read and -timeout:write.")
  fReadTimeout  := cmdline.Duration ("timeout:read",  strToDur(coalesce(os.Getenv("HP_TIMEOUT_READ"), "5s")),               "The read timeout for client connections.")
  fWriteTimeout := cmdline.Duration ("timeout:write", strToDur(coalesce(os.Getenv("HP_TIMEOUT_WRITE"), "5s")),              "The write timeout for client connections.")
  fCacheTimeout := cmdline.Duration ("timeout:cache", strToDur(coalesce(os.Getenv("HP_TIMEOUT_CACHE"), "30s")),             "The timeout for cached service providers. This should not be significantly larger than the backend's expiration.")
  fOptimize     := cmdline.Bool     ("optimize",      strToBool(os.Getenv("HP_OPTIMIZE")),                                  "Optimize data transfer, if possible, by enabling zero-copy transfer.")
  fDebug        := cmdline.Bool     ("debug",         strToBool(os.Getenv("HP_DEBUG")),                                     "Enable debugging mode.")
  fStack        := cmdline.Bool     ("debug:stack",   strToBool(os.Getenv("HP_DEBUG_STACK")),                               "Enable stack debugging mode.")
  fVerbose      := cmdline.Bool     ("verbose",       strToBool(os.Getenv("HP_VERBOSE")),                                   "Enable verbose debugging mode.")
  cmdline.Var    (&proxyRoutes,      "route",                                                                               "Add a proxy route for the specified service as: 'listen_port=(host:port,...|service)'. Use this flag repeatedly for multiple routes.")
  cmdline.Parse(os.Args[1:])
  
  if r := os.Getenv("HP_ROUTES"); r != "" {
    for _, e := range strings.Split(r, ";") {
      proxyRoutes = append(proxyRoutes, strings.TrimSpace(e))
    }
  }
  if len(proxyRoutes) < 1 {
    fmt.Println("* * * No routes defined; use -route 'listen_port=(host:port,...|service)'")
    os.Exit(-1)
  }
  
  debug.DEBUG = *fDebug
  debug.VERBOSE = *fVerbose
  if debug.DEBUG {
    fmt.Println("-----> Debugging mode enabled")
  }
  
  hostname, err := os.Hostname()
  if err != nil {
    hostname = "unknown"
  }
  fmt.Printf("-----> Hostname: %v\n", hostname)
  
  digest := sha1.Sum([]byte(rand.HardwareKey()))
  instance := fmt.Sprintf("%x", digest[:])
  fmt.Printf("-----> Instance key: %v (%v)\n", instance, rand.HardwareAddr())
  
  loggers := make([]alt.Target, 0)
  if *fSentry != "" {
    fmt.Println("-----> Alerting via Sentry")
    logger, err := sentry.New(*fSentry, alt.LEVEL_ERROR)
    if err != nil {
      panic(err)
    }
    loggers = append(loggers, logger)
  }
  
  alt.Init(alt.Config{
    Debug: true,
    Verbose: debug.VERBOSE,
    Name: "percolator",
    Tags: map[string]interface{}{alt.TAG_HOSTNAME:hostname, alt.TAG_ENVIRON:*fEnviron},
    Targets: loggers,
  })
  
  defer func() {
    if r := recover(); r != nil {
      alt.Fatalf("PERCOLATOR IS CRASHING (%v): %v", hostname, r)
      panic(r) // actually panic
    }
  }()
  
  if *fInflux != "" {
    fmt.Printf("-----> Reporting metrics to InfluxDB: %v (%v)\n", *fInflux, *fEnviron)
    metrics.RegisterDebugGCStats(metrics.DefaultRegistry)
    metrics.RegisterRuntimeMemStats(metrics.DefaultRegistry)
    go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, time.Second * 5, fmt.Sprintf("http://%s", *fInflux), "hirepurpose", "", "", map[string]string{"environ": *fEnviron, "host": hostname, "instance": instance})
  }
  
  disc, err := discovery.New(*fDomain, *fDiscovery)
  if err != nil {
    panic(err)
  }
  
  if *fCacheTimeout > 0 {
    if disc != nil {
      disc = discovery.NewCache(disc, *fCacheTimeout)
    }
  }
  
  var routes []*route.Route
  for _, e := range proxyRoutes {
    r, err := route.Parse(e)
    if err != nil {
      panic(err)
    }
    if r.Service && disc == nil {
      panic(fmt.Errorf("No discovery service is defined but a service is used in route: %v", e))
    }
    routes = append(routes, r)
  }
  
  if *fStack {
    fmt.Println("-----> Stack debugging enabled; use ^C to dump routines")
    debug.DumpRoutinesOnInterrupt()
  }
  if *fOptimize {
    fmt.Println("-----> Enabling OS-specific optimizations")
    panic("OS-specific optimizations are broken!")
  }
  if *fIOTimeout > 0 {
    *fReadTimeout = *fIOTimeout
    *fWriteTimeout = *fIOTimeout
  }
  
  if *fProfile != "" && *fProfile != "none" {
    fmt.Printf("-----> Profiling via pprof: %v\n", *fProfile)
    go func() {
      alt.Errorf("* * * Could not profile: %v", http.ListenAndServe(*fProfile, nil))
    }()
  }
  
  svc := service.New(service.Config{
    Name:         "percolator",
    Instance:     instance,
    Discovery:    disc,
    Routes:       routes,
    ZeroCopy:     *fOptimize,
    ReadTimeout:  *fReadTimeout,
    WriteTimeout: *fWriteTimeout,
  })
  
  panic(svc.Run())
}

/**
 * String to bool
 */
func strToBool(s string) bool {
  return strings.EqualFold(s, "t") || strings.EqualFold(s, "true") || strings.EqualFold(s, "y") || strings.EqualFold(s, "yes")
}

/**
 * String to duration
 */
func strToDur(s string) time.Duration {
  d, err := time.ParseDuration(s)
  if err != nil {
    panic(err)
  }
  return d
}

/**
 * Return the first non-empty string from those provided
 */
func coalesce(v... string) string {
  for _, e := range v {
    if e != "" {
      return e
    }
  }
  return ""
}

/**
 * Flag string list
 */
type flagList []string

/**
 * Set a flag
 */
func (s *flagList) Set(v string) error {
  *s = append(*s, v)
  return nil
}

/**
 * Describe
 */
func (s *flagList) String() string {
  return fmt.Sprintf("%+v", *s)
}
