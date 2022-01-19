package main

import (
	"errors"
	"fmt"
	"github.com/gavv/httpexpect/v2"
	"github.com/gorilla/mux"
	"github.com/valyala/fasthttp"
	"net/http"
	"testing"
	"time"
)

//TestGetCurrentNode - to test the basic functional of balancer - to deliver response from correct node
func TestGetCurrentNode(t *testing.T) {
	setupTestServers()

	t.Setenv("NODES", `[ {"node":"127.0.0.1:9001", "is_tls":false},
									{"node":"127.0.0.1:9002", "is_tls":false},
									{"node":"127.0.0.1:9003", "is_tls":false},
									{"node":"127.0.0.1:9004", "is_tls":false},
									{"node":"127.0.0.1:9005", "is_tls":false},
									{"node":"127.0.0.1:9006", "is_tls":false},
									{"node":"127.0.0.1:9007", "is_tls":false},
									{"node":"127.0.0.1:9008", "is_tls":false}]`)
	t.Setenv("TLS", "OFF")
	t.Setenv("SELECT_NODE_PERIOD", fmt.Sprintf("%d", 3))
	t.Setenv("CLEAN_STATS_PERIOD", fmt.Sprintf("%d", 2))
	t.Setenv("NODE_TIMEOUT", "3000")
	t.Setenv("HOST_PORT", "127.0.0.1:8000")
	t.Setenv("NODE_TIMEOUT", "100000")

	for _, engine := range []string{"THUNDER", "FAST_HTTP", "STANDARD_HTTP"} {
		t.Logf("Testing ENGINE=%s", engine)
		t.Setenv("ENGINE", engine)
		cfg := newConfigFromEnv()
		s := newServer(cfg)
		s.start()
		time.Sleep(time.Second)

		e := httpexpect.New(t, fmt.Sprintf("http://127.0.0.1:8000"))
		for i := 1; i <= 10; i++ {
			if i == 10 {
				e.GET("/").Expect().Body().Equal("1")
			} else {
				e.GET("/").Expect().Body().Equal(fmt.Sprintf("%d", 1+(i-1)/3))
			}
		}
		s.stop()

	}

}

//BenchmarkGetCurrentNode - to be sure that getCurrentNode stays zero-allocation
func BenchmarkGetCurrentNode(b *testing.B) {
	b.Setenv("NODES", `[{"node":"127.0.0.1:9000", "is_tls":false},
									{"node":"127.0.0.1:9001", "is_tls":false},
									{"node":"127.0.0.1:9002", "is_tls":false},
									{"node":"127.0.0.1:9003", "is_tls":false}]`)
	b.Setenv("TLS", "OFF")
	b.Setenv("SELECT_NODE_PERIOD", fmt.Sprintf("%d", 1))
	b.Setenv("CLEAN_STATS_PERIOD", fmt.Sprintf("%d", 1))
	b.Setenv("ENGINE", "STANDARD_HTTP")
	b.Setenv("HOST_PORT", "127.0.0.1:8000")
	b.Setenv("NODE_TIMEOUT", "3000")

	cfg := newConfigFromEnv()
	h := NewBalancer(cfg)

	var currentNode int32
	var recalculationsCountState int32
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		currentNode, recalculationsCountState = h.(*balancer).getCurrentNode()
		h.(*balancer).updateNodeStat(100, currentNode, recalculationsCountState)
	}
}

//TestPreventSlowNodeToBeChosen - to test the timeouts / will the next node be chosen correctly
func TestPreventSlowNodeToBeChosen(t *testing.T) {
	setupTestServers()

	t.Setenv("NODES", `[ {"node":"127.0.0.1:9001", "is_tls":false},
									{"node":"127.0.0.1:9002", "is_tls":false},
									{"node":"127.0.0.1:9003", "is_tls":false},
									{"node":"127.0.0.1:9004", "is_tls":false},
									{"node":"127.0.0.1:9005", "is_tls":false},
									{"node":"127.0.0.1:9006", "is_tls":false},
									{"node":"127.0.0.1:9007", "is_tls":false},
									{"node":"127.0.0.1:9008", "is_tls":false}]`)
	t.Setenv("TLS", "OFF")
	t.Setenv("SELECT_NODE_PERIOD", fmt.Sprintf("%d", 1))
	t.Setenv("CLEAN_STATS_PERIOD", fmt.Sprintf("%d", 200))

	t.Setenv("HOST_PORT", "127.0.0.1:8000")
	t.Setenv("NODE_TIMEOUT", "3000")

	for _, engine := range []string{"THUNDER", "FAST_HTTP", "STANDARD_HTTP"} {
		t.Logf("Testing ENGINE=%s", engine)
		t.Setenv("ENGINE", engine)
		cfg := newConfigFromEnv()

		s := Server{
			cfg: cfg,
		}
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
			go s.engine.(*fasthttp.Server).ListenAndServe(s.cfg.hostAndPort)
		case "STANDARD_HTTP":
			s.engine = &http.Server{Addr: s.cfg.hostAndPort, Handler: h}
			go s.engine.(*http.Server).ListenAndServe()

		default:
			panic(errors.New("define the THUNDER_ENGINE env variable (THUNDER, FAST_HTTP or STANDARD_HTTP)"))
		}
		time.Sleep(time.Second)
		e := httpexpect.New(t, fmt.Sprintf("http://127.0.0.1:8000"))
		slowNodeAlreadyUsed := false
		for i := 1; i <= 20; i++ {

			if i == 2 {
				e.GET("/").Expect().Status(http.StatusGatewayTimeout)
			} else {
				e.GET("/").Expect().Status(http.StatusOK)
			}
			if h.(*balancer).d.currentNode.get() == 1 {

				if !slowNodeAlreadyUsed {
					slowNodeAlreadyUsed = true
					t.Logf("slow node has been chosen")
				} else {
					t.Fatalf("slow node has been selected again")
				}
			}
		}
		s.stop()

	}

}

//setupTestServers - to create dummy nodes
func setupTestServers() {
	for i := 1; i <= 8; i++ {

		func(number int) {
			r := mux.NewRouter()
			r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

				if number == 2 {
					time.Sleep(4 * time.Second)
				}

				time.Sleep(time.Duration(i*100) * time.Millisecond)
				w.Write([]byte(fmt.Sprintf("%d", number)))
			})

			srv := &http.Server{
				Handler:      r,
				Addr:         fmt.Sprintf("127.0.0.1:%d", 9000+number),
				WriteTimeout: 15 * time.Second,
				ReadTimeout:  15 * time.Second,
			}
			go srv.ListenAndServe()
		}(i)
	}
	time.Sleep(time.Second)
}
