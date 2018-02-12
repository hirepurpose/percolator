package service

import (
  "io"
  "fmt"
  "net"
  "time"
  "crypto/tls"
  "sync/atomic"
  
  "perc/route"
  "perc/discovery"
)

import (
  "golang.org/x/net/trace"
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
  "github.com/rcrowley/go-metrics"
)

const (
  paramTLS  = "tls"
  paramHTTP = "http"
)

var (
  proxyConnRate metrics.Meter
  proxyResolveTimer metrics.Timer
  proxyResolveError metrics.Meter
  proxyLatencyTimer metrics.Timer
  proxyConnError metrics.Meter
  proxyXferError metrics.Meter
  proxyBytesReadRate metrics.Meter
  proxyBytesWriteRate metrics.Meter
)

func init() {
  proxyConnRate = metrics.NewMeter()
  metrics.Register("percolator.proxy.conn.rate", proxyConnRate)
  proxyConnError = metrics.NewMeter()
  metrics.Register("percolator.proxy.conn.error", proxyConnError)
  proxyXferError = metrics.NewMeter()
  metrics.Register("percolator.proxy.xfer.error", proxyXferError)
  proxyResolveError = metrics.NewMeter()
  metrics.Register("percolator.proxy.resolve.error", proxyResolveError)
  proxyResolveTimer = metrics.NewTimer()
  metrics.Register("percolator.proxy.resolve.latency", proxyResolveTimer)
  proxyLatencyTimer = metrics.NewTimer()
  metrics.Register("percolator.proxy.conn.latency", proxyLatencyTimer)
  proxyBytesReadRate = metrics.NewMeter()
  metrics.Register("percolator.proxy.bytes.read.rate", proxyBytesReadRate)
  proxyBytesWriteRate = metrics.NewMeter()
  metrics.Register("percolator.proxy.bytes.write.rate", proxyBytesWriteRate)
}

// Service stats
type Stats struct {
  OpenConnections           int64             `json:"open_conns"`
  TotalConnections          int64             `json:"total_conns"`
  BytesTransferred          int64             `json:"bytes_xfer"`
  TotalConnectionsByRoute   map[string]int64  `json:"total_conns_by_route"`
  RunningWorkers            int64             `json:"io_workers"`
}

// Service config
type Config struct {
  Name          string
  Instance      string
  Discovery     discovery.Service
  Routes        []*route.Route
  ConnTimeout   time.Duration
  ReadTimeout   time.Duration
  WriteTimeout  time.Duration
  Debug         bool
}

// An API service
type Service struct {
  name            string
  instance        string
  discovery       discovery.Service
  routes          []*route.Route
  cto, rto, wto   time.Duration
  debug           bool
  //
  copyOpen        int64
  handlerOpen     int64
  handlerTotal    int64
  handlerXfer     int64
  handlerByRoute  *cmap
  handlerUpdate   chan<- entry
}

// Create a new service
func New(conf Config) *Service {
  m := newCmap()
  return &Service{
    conf.Name, conf.Instance, conf.Discovery, conf.Routes, conf.ConnTimeout, conf.ReadTimeout, conf.WriteTimeout, conf.Debug,
    0, 0, 0, 0, m, m.Put(),
  }
}

// How many connections are we currently handling
func (s *Service) Stats() Stats {
  return Stats{
    OpenConnections:atomic.LoadInt64(&s.handlerOpen),
    TotalConnections:atomic.LoadInt64(&s.handlerTotal),
    BytesTransferred:atomic.LoadInt64(&s.handlerXfer),
    TotalConnectionsByRoute:s.handlerByRoute.Copy(),
    RunningWorkers:atomic.LoadInt64(&s.copyOpen),
  }
}

// Handle requests forever
func (s *Service) Run() error {
  var err error
  
  l := make([]net.Listener, len(s.routes))
  for i, e := range s.routes {
    l[i], err = net.Listen("tcp", e.Listen)
    if err != nil {
      return err
    }
  }
  
  errs := make(chan error)
  for i, e := range l {
    r := s.routes[i]
    fmt.Printf("-----> Serving requests on: %s\n", r.Detail())
    go func(r *route.Route, l net.Listener){
      for {
        conn, err := l.Accept()
        if err != nil {
          alt.Errorf("service: Could not accept: %v", err)
          continue
        }else{
          proxyConnRate.Mark(1)
          go s.handle(r, conn.(*net.TCPConn))
        }
      }
    }(r, e)
  }
  
  return <- errs
}

// Handle a request for a particular route
func (s *Service) handle(r *route.Route, c *net.TCPConn) {
  var p net.Conn
  var err error
  
  var caddr string
  if h, _, err := net.SplitHostPort(c.RemoteAddr().String()); err == nil {
    caddr = h
  }else{
    caddr = "<unknown>"
  }
  
  var tr trace.Trace
  if s.debug {
    var b string
    if r.Service {
      b = r.Any().String()
    }else{
      b = "<next>"
    }
    tr = trace.New("perc.Service", fmt.Sprintf("%v -> %v", caddr, b))
    defer tr.Finish()
  }
  
  atomic.AddInt64(&s.handlerOpen, 1)
  atomic.AddInt64(&s.handlerTotal, 1)
  defer atomic.AddInt64(&s.handlerOpen, -1)
  
  if debug.VERBOSE {
    alt.Debugf("%v: Accepted connection", c.RemoteAddr())
  }
  if tr != nil {
    tr.LazyPrintf("Accepted connection: %v", c.RemoteAddr())
  }
  
  defer func() {
    if c != nil {
      err = c.Close()
      if err != nil {
        alt.Errorf("service: %v: Could not close client: %v\n", c.RemoteAddr(), err)
        if tr != nil {
          tr.LazyPrintf("%v: Could not close client: %v\n", c.RemoteAddr(), err)
          tr.SetError()
        }
      }
    }
  }()
  
  start := time.Now()
  
  var addr string
  var backend route.Backend
  if r.Service {
    if s.discovery == nil {
      proxyResolveError.Mark(1)
      if debug.VERBOSE {
        alt.Errorf("service: Discovery not available")
      }
      if tr != nil {
        tr.LazyPrintf("Discovery not available")
        tr.SetError()
      }
      return
    }
    backend = r.Any()
    addr, err = s.discovery.LookupProvider(backend.Addr)
    if err != nil {
      proxyResolveError.Mark(1)
      if debug.VERBOSE {
        alt.Errorf("service: Could not discover service: %v: %v", r.String(), err)
      }
      if tr != nil {
        tr.LazyPrintf("Could not discover service: %v: %v", r.String(), err)
        tr.SetError()
      }
      return
    }
    s.handlerUpdate <- entry{backend.String(), 1, caddr}
  }else{
    backend = r.Next()
    addr = backend.Addr
    s.handlerUpdate <- entry{addr, 1, caddr}
  }
  
  proxyResolveTimer.Update(time.Since(start))
  
  if debug.VERBOSE {
    alt.Debugf("%v: Proxying to backend: %v (%v)", c.RemoteAddr(), addr, backend)
  }
  
  defer func() {
    if p != nil {
      err = p.Close()
      if err != nil {
        alt.Errorf("service: %v: Could not close backend: %v\n", p.RemoteAddr(), err)
        if tr != nil {
          tr.LazyPrintf("%v: Could not close backend: %v\n", p.RemoteAddr(), err)
          tr.SetError()
        }
      }
    }
  }()
  
  start = time.Now()
  
  d := &net.Dialer{Timeout:s.cto}
  if name, ok := backend.Params[paramTLS]; ok {
    if tr != nil {
      tr.LazyPrintf("%v: Proxying to backend: %v (%v) via TLS (SNI: %v)", c.RemoteAddr(), addr, backend, name)
    }
    p, err = tls.DialWithDialer(d, "tcp", addr, &tls.Config{ServerName:name})
  }else{
    if tr != nil {
      tr.LazyPrintf("%v: Proxying to backend: %v (%v)", c.RemoteAddr(), addr, backend)
    }
    p, err = d.Dial("tcp", addr)
  }
  if err != nil {
    proxyConnError.Mark(1)
    if debug.VERBOSE {
      alt.Debugf("service: %v: Could not connect to backend: %v (%v): %v", c.RemoteAddr(), addr, backend, err)
    }
    if tr != nil {
      tr.LazyPrintf("%v: Could not connect to backend: %v (%v): %v", c.RemoteAddr(), addr, backend, err)
      tr.SetError()
    }
    return
  }
  
  proxyLatencyTimer.Update(time.Since(start))
  
  rerrs := make(chan error, 1)
  werrs := make(chan error, 1)
  
  go s.copyGeneric(c, p, tr, proxyBytesReadRate, rerrs)
  go s.copyGeneric(p, c, tr, proxyBytesWriteRate, werrs)
  
  var ok bool
  select {
    case err, ok = <- rerrs:
    case err, ok = <- werrs:
  }
  if ok && err != io.EOF {
    proxyXferError.Mark(1)
    if debug.VERBOSE {
      alt.Debugf("service: %v -> %v (%v): Could not proxy: %v\n", c.RemoteAddr(), p.RemoteAddr(), backend, err)
    }
    if tr != nil {
      tr.LazyPrintf("%v -> %v (%v): Could not proxy: %v\n", c.RemoteAddr(), p.RemoteAddr(), backend, err)
      tr.SetError()
    }
  }
  
  if debug.VERBOSE {
    alt.Debugf("%v: Connection will end: %v (%v)", c.RemoteAddr(), addr, backend)
  }
  if tr != nil {
    tr.LazyPrintf("%v: Connection will end: %v (%v)", c.RemoteAddr(), addr, backend)
  }
}

// Handling copying from a source to destination connection
func (s *Service) copyGeneric(dst, src net.Conn, tr trace.Trace, xfer metrics.Meter, errs chan<- error) {
  var copied int64
  
  atomic.AddInt64(&s.copyOpen, 1)
  defer func(){
    atomic.AddInt64(&s.copyOpen, -1)
    close(errs)
  }()
  
  buf := make([]byte, 32 * 1024)
  for {
    nr, er := src.Read(buf)
    xfer.Mark(int64(nr)) // read side is instrumented
    atomic.AddInt64(&s.handlerXfer, int64(nr))
    if s.rto > 0 { // read deadline on src only
      src.SetReadDeadline(time.Now().Add(s.rto))
    }
    if s.wto > 0 { // write deadline on src only
      src.SetWriteDeadline(time.Now().Add(s.wto))
    }
    if nr > 0 {
      nw, ew := dst.Write(buf[0:nr])
      if nw > 0 {
        copied += int64(nw)
      }
      if ew != nil {
        errs <- ew
        break
      }
      if nr != nw {
        errs <- io.ErrShortWrite
        break
      }
    }
    if er != nil {
      if er != io.EOF {
        errs <- er
      }
      break
    }
  }
  
  if debug.VERBOSE && copied > 0 {
    alt.Debugf("%v -> %v: copied %d", src.RemoteAddr(), dst.RemoteAddr(), copied)
  }
  if tr != nil {
    tr.LazyPrintf("%v -> %v: copied %d", src.RemoteAddr(), dst.RemoteAddr(), copied)
  }
}
