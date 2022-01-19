package main

import (
	"crypto/rand"
	"crypto/tls"
	"log"
	"net"
)

//ThunderEngineConfig data for thunder engine
type ThunderEngineConfig struct {
	hostAndPort string
	isTLS       bool
	tlsCert     string
	tlsKey      string
}

//ThunderEngine contains host/port and port listener
type ThunderEngine struct {
	addr     string
	listener net.Listener
}

//Shutdown closes port and stops listener
func (e *ThunderEngine) Shutdown() {
	_ = e.listener.Close()
}

//ListenAndServe opens port and starts requests handling
func (e *ThunderEngine) ListenAndServe(handlerFunc func(conn net.Conn, err error), cfg ThunderEngineConfig) {

	go func() {
		if cfg.isTLS {
			cert, err := tls.LoadX509KeyPair(cfg.tlsCert, cfg.tlsKey)
			if err != nil {
				log.Fatalf("server: loadkeys: %s", err)
			}

			tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
			tlsConfig.Rand = rand.Reader
			e.listener, err = tls.Listen("tcp", e.addr, &tlsConfig)

			if err != nil {
				log.Fatalf("server: listen: %s", err)
			}
		} else {
			var err error
			e.listener, err = net.Listen("tcp", cfg.hostAndPort)
			if err != nil {
				log.Fatalf("server: listen: %s", err)
			}
		}

		for {
			go handlerFunc(e.listener.Accept())
		}
	}()
}
