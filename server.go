package main

import (
	"context"
	"errors"
	"github.com/valyala/fasthttp"
	"net/http"
)

//Server stores config and initialized engine
type Server struct {
	cfg    Config
	engine interface{}
}

func newServer(cfg Config) Server {
	return Server{cfg: cfg}
}

//start begins listening the port according to engine
func (s *Server) start() {
	h := NewBalancer(s.cfg)

	switch s.cfg.engine {
	case "THUNDER":
		s.engine = &ThunderEngine{addr: s.cfg.hostAndPort}
		s.engine.(*ThunderEngine).ListenAndServe(h.getThunderHandlerFunc(), ThunderEngineConfig{
			hostAndPort: s.cfg.hostAndPort,
			isTLS:       s.cfg.tlsUsed,
			tlsCert:     s.cfg.tlsCert,
			tlsKey:      s.cfg.tlsKey,
		})
	case "FAST_HTTP":
		s.engine = &fasthttp.Server{Handler: h.getFastHTTPHandler()}
		if s.cfg.tlsUsed {
			go s.engine.(*fasthttp.Server).ListenAndServeTLS(s.cfg.hostAndPort, s.cfg.tlsCert, s.cfg.tlsKey)
		} else {
			go s.engine.(*fasthttp.Server).ListenAndServe(s.cfg.hostAndPort)
		}
	case "STANDARD_HTTP":
		s.engine = &http.Server{Addr: s.cfg.hostAndPort, Handler: h}
		if s.cfg.tlsUsed {
			go s.engine.(*http.Server).ListenAndServeTLS(s.cfg.tlsCert, s.cfg.tlsKey)
		} else {
			go s.engine.(*http.Server).ListenAndServe()
		}

	default:
		panic(errors.New("define the THUNDER_ENGINE env variable (THUNDER, FAST_HTTP or STANDARD_HTTP)"))
	}
}

//stop performs shutting down of server and closes port
func (s *Server) stop() {
	switch s.cfg.engine {
	case "THUNDER":
		s.engine.(*ThunderEngine).Shutdown()
	case "FAST_HTTP":
		s.engine.(*fasthttp.Server).Shutdown()
	case "STANDARD_HTTP":
		s.engine.(*http.Server).Shutdown(context.TODO())

	default:
		panic(errors.New("define the THUNDER_ENGINE env variable (THUNDER, FAST_HTTP or STANDARD_HTTP)"))
	}
}
