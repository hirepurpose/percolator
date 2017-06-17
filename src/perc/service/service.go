package service

import (
  "io"
  "fmt"
  "net"
  "time"
  "strings"
  "perc/route"
  "perc/discovery"
  "os"
  "os/signal"
  "sync/atomic"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
  "github.com/rcrowley/go-metrics"
)

var (
  copyOpen    int64
  handlerOpen int64
)

var (
  proxyConnRate metrics.Meter
  proxyBytesReadRate metrics.Meter
  proxyBytesWriteRate metrics.Meter
)

func init() {
  proxyConnRate = metrics.NewMeter()
  metrics.Register("percolator.proxy.conn.rate", proxyConnRate)
  proxyBytesReadRate = metrics.NewMeter()
  metrics.Register("percolator.proxy.bytes.read.rate", proxyBytesReadRate)
  proxyBytesWriteRate = metrics.NewMeter()
  metrics.Register("percolator.proxy.bytes.write.rate", proxyBytesWriteRate)
}

/**
 * Service config
 */
type Config struct {
  Name          string
  Instance      string
  Discovery     discovery.Service
  Routes        []*route.Route
  ReadTimeout   time.Duration
  WriteTimeout  time.Duration
  Debug         bool
}

/**
 * An API service
 */
type Service struct {
  name      string
  instance  string
  discovery discovery.Service
  routes    []*route.Route
  rto, wto  time.Duration
  debug     bool
}

/**
 * Create a new service
 */
func New(conf Config) *Service {
  return &Service{conf.Name, conf.Instance, conf.Discovery, conf.Routes, conf.ReadTimeout, conf.WriteTimeout, conf.Debug}
}

/**
 * Handle requests forever
 */
func (s *Service) Run() error {
  var err error
  
  if s.debug {
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, os.Interrupt)
    go func() {
      for range sig {
        fmt.Printf("service: Currently running handlers: %d, I/O workers: %d\n", atomic.LoadInt64(&handlerOpen), atomic.LoadInt64(&copyOpen))
      }
    }()
  }
  
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
    fmt.Printf("-----> Serving requests on: %v\n", r)
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

/**
 * Handle a request for a particular route
 */
func (s *Service) handle(r *route.Route, c *net.TCPConn) {
  var b *net.TCPConn
  var err error
  
  atomic.AddInt64(&handlerOpen, 1)
  defer atomic.AddInt64(&handlerOpen, -1)
  
  if debug.VERBOSE {
    alt.Debugf("%v: Accepted connection", c.RemoteAddr())
  }
  
  defer func() {
    if c != nil {
      err = c.Close()
      if err != nil {
        alt.Errorf("service: %v -> %v: Could not close client: %v\n", c.RemoteAddr(), b.RemoteAddr(), err)
      }
    }
    if b != nil {
      err = b.Close()
      if err != nil {
        alt.Errorf("service: %v -> %v: Could not close backend: %v\n", err)
      }
    }
  }()
  
  var backend, addr string
  if r.Service {
    if s.discovery == nil {
      alt.Errorf("service: Discovery not available")
      return
    }
    backend = r.Backends[0]
    addr, err = s.discovery.LookupProvider(backend)
    if err != nil {
      alt.Errorf("service: Could not discover service: %v: %v", strings.Join(r.Backends, ", "), err)
      return
    }
  }else{
    addr = r.NextBackend()
  }
  
  if debug.VERBOSE {
    alt.Debugf("%v: Proxying to backend: %v (%v)", c.RemoteAddr(), addr, backend)
  }
  
  d := net.Dialer{Timeout: time.Second * 5}
  p, err := d.Dial("tcp", addr)
  if err != nil {
    alt.Errorf("service: %v: Could not connect to backend: %v (%v)", c.RemoteAddr(), addr, backend)
    return
  }
  
  b = p.(*net.TCPConn)
  rerrs := make(chan error, 1)
  werrs := make(chan error, 1)
  
  go s.copyGeneric(c, b, proxyBytesReadRate, rerrs)
  go s.copyGeneric(b, c, proxyBytesWriteRate, werrs)
  
  var ok bool
  select {
    case err, ok = <- rerrs:
    case err, ok = <- werrs:
  }
  if ok && err != io.EOF {
    alt.Errorf("service: %v -> %v (%v): Could not proxy: %v\n", c.RemoteAddr(), b.RemoteAddr(), backend, err)
  }
  
  if debug.VERBOSE {
    alt.Debugf("%v: Connection will end: %v (%v)", c.RemoteAddr(), addr, backend)
  }
}

/**
 * Handling copying from a source to destination connection
 */
func (s *Service) copyGeneric(dst, src *net.TCPConn, xfer metrics.Meter, errs chan<- error) {
  var copied int64
  
  atomic.AddInt64(&copyOpen, 1)
  defer func(){
    atomic.AddInt64(&copyOpen, -1)
    close(errs)
  }()
  
  buf := make([]byte, 32 * 1024)
  for {
    nr, er := src.Read(buf)
    xfer.Mark(int64(nr)) // read side is instrumented
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
    alt.Debugf("%v -> %v: copied <gen> %d", src.RemoteAddr(), dst.RemoteAddr(), copied)
  }
}
