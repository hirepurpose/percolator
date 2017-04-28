package service

import (
  "io"
  "fmt"
  "net"
  "time"
  "perc/route"
  "perc/discovery"
)

import (
  "github.com/bww/go-alert"
  "github.com/bww/go-util/debug"
  "github.com/rcrowley/go-metrics"
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
  ZeroCopy      bool
  ReadTimeout   time.Duration
  WriteTimeout  time.Duration
}

/**
 * An API service
 */
type Service struct {
  name      string
  instance  string
  discovery discovery.Service
  routes    []*route.Route
  optimize  bool
  rto, wto  time.Duration
}

/**
 * Create a new service
 */
func New(conf Config) *Service {
  return &Service{conf.Name, conf.Instance, conf.Discovery, conf.Routes, conf.ZeroCopy, conf.ReadTimeout, conf.WriteTimeout}
}

/**
 * Handle requests forever
 */
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
  
  var addr string
  if r.Service {
    if s.discovery == nil {
      alt.Errorf("service: Discovery not available")
      return
    }
    addr, err = s.discovery.LookupProvider(r.Backends[0])
    if err != nil {
      alt.Errorf("service: Could not discover service: %v", err)
      return
    }
  }else{
    addr = r.NextBackend()
  }
  
  if debug.VERBOSE {
    alt.Debugf("%v: Proxying to backend: %v", c.RemoteAddr(), addr)
  }
  
  p, err := net.Dial("tcp", addr)
  if err != nil {
    alt.Errorf("service: %v: Could not connect to backend: %v", c.RemoteAddr(), addr)
    return
  }
  
  b = p.(*net.TCPConn)
  errs := make(chan error, 2)
  
  if s.optimize {
    go s.copyOptimized(c, b, proxyBytesReadRate, errs)
    go s.copyOptimized(b, c, proxyBytesWriteRate, errs)
  }else{
    go s.copyGeneric(c, b, proxyBytesReadRate, errs)
    go s.copyGeneric(b, c, proxyBytesWriteRate, errs)
  }
  
  err = <- errs
  if err != io.EOF {
    alt.Errorf("service: %v -> %v: Could not proxy: %v\n", c.RemoteAddr(), b.RemoteAddr(), err)
  }
  
  if debug.VERBOSE {
    alt.Debugf("%v: Connection will end", c.RemoteAddr())
  }
}

/**
 * Handling copying from a source to destination connection
 */
func (s *Service) copyGeneric(dst, src *net.TCPConn, xfer metrics.Meter, errs chan<- error) {
  var copied int64
  
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
  
  if debug.VERBOSE {
    alt.Debugf("%v -> %v: copied <gen> %d", src.RemoteAddr(), dst.RemoteAddr(), copied)
  }
  errs <- io.EOF
}

/**
 * Handling transfering data from a source to destination connection. Try
 * to avoid copying data.
 */
func (s *Service) copyOptimized(dst, src *net.TCPConn, xfer metrics.Meter, errs chan<- error) {
  var err error
  
  err = src.SetKeepAlive(false)
  if err != nil {
    errs <- err; return
  }
  
  err = dst.SetKeepAlive(false)
  if err != nil {
    errs <- err; return
  }
  
  f, err := src.File()
  if err != nil {
    errs <- err; return
  }
  
  n, err := dst.ReadFrom(f)
  if err != nil {
    errs <- err; return
  }
  
  xfer.Mark(n)
  
  if debug.VERBOSE {
    alt.Debugf("%v -> %v: copied <opt> %d", src.RemoteAddr(), dst.RemoteAddr(), n)
  }
  errs <- io.EOF
}
