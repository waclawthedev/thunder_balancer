package main

import (
	"crypto/tls"
	"errors"
	"github.com/valyala/fasthttp"
	"net"
	"net/http"
	"time"
)

//Balancer describes the methods for different engines
type Balancer interface {
	getFastHTTPHandler() func(ctx *fasthttp.RequestCtx)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	getThunderHandlerFunc() func(conn net.Conn, err error)
}

type balancer struct {
	d   *DataSet
	cfg *Config
}

//NewBalancer returns Balancer
func NewBalancer(c Config) Balancer {
	d := newDataSetFromConfig(c)
	return &balancer{d: d, cfg: &c}
}

//chooseNewNode assigns new node based on response time in stats
func chooseNewNode(d *DataSet, recalculateState int32) {
	//check if recalculation was not performed before this call
	if recalculateState != d.recalculationsCount.get() {
		return
	}
	d.chooseNewNodeMutex.Lock()
	defer d.chooseNewNodeMutex.Unlock()
	//check if recalculation was not performed while waiting for mutex unlock
	if recalculateState != d.recalculationsCount.get() {
		return
	}
	//cycle the counter
	if d.recalculationsCount.add(1) > 1000000 {
		d.recalculationsCount.set(0)
	}

	var newNodeCandidate int32
	var newNodeCandidateAvg int64
	var i int32
	for i = 0; i < d.nodesAmount; i++ {

		if i == 0 {
			newNodeCandidate = 0
			newNodeCandidateAvg = d.nodes[0].overallResponseTime.get() / d.nodes[0].requestsCount.get()
		} else {

			if avg := d.nodes[i].overallResponseTime.get() / d.nodes[i].requestsCount.get(); avg < newNodeCandidateAvg {
				newNodeCandidateAvg = avg
				newNodeCandidate = i
			}
		}

	}
	d.currentNode.set(newNodeCandidate)

}

//getFastHTTPHandler return handler for fasthttp engine
func (h *balancer) getFastHTTPHandler() func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		currentNode, recalculationsCountState := h.getCurrentNode()
		requestServed := false

		resp := fasthttp.Response{}

		client := fasthttp.Client{Dial: func(addr string) (net.Conn, error) {
			return fasthttp.Dial(h.d.nodes[currentNode].ipAndPort)
		}}

		startTime := time.Now()
		err := client.DoTimeout(&ctx.Request, &resp, time.Duration(h.cfg.nodeTimeout)*time.Millisecond)
		if err != nil {
			if !errors.Is(err, fasthttp.ErrTimeout) {
				requestServed = true
			}
		} else {
			requestServed = true
		}

		h.updateNodeStat(time.Since(startTime).Milliseconds(), currentNode, recalculationsCountState)

		if !requestServed {
			ctx.Response.Header.SetStatusCode(http.StatusGatewayTimeout)
		} else {
			resp.CopyTo(&ctx.Response)
		}
	}
}

//getThunderHandlerFunc returns handler function for Thunder engine
func (h *balancer) getThunderHandlerFunc() func(conn net.Conn, err error) {
	return func(conn net.Conn, err error) {
		if err != nil {
			return
		}
		currentNode, recalculationsCountState := h.getCurrentNode()
		requestServed := false
		timeout := time.Duration(h.cfg.nodeTimeout) * time.Millisecond

		if h.cfg.tlsUsed {
			_, ok := conn.(*tls.Conn)
			if !ok {
				return
			}
		}

		node, _ := net.Dial("tcp", h.d.nodes[currentNode].ipAndPort)

		buf := make([]byte, 512)

		for {
			n, err := conn.Read(buf)
			_ = node.SetWriteDeadline(time.Now().Add(timeout))
			_, _ = node.Write(buf[:n])
			if n < 512 || err != nil {
				break
			}

		}

		responseReadStarted := false
		_ = node.SetReadDeadline(time.Now().Add(timeout))
		startTime := time.Now()
		for {
			nR, rErr := node.Read(buf)
			if rErr != nil {
				_ = node.Close()
				requestServed = false
				break
			}

			if !responseReadStarted {
				responseReadStarted = true
				_ = node.SetReadDeadline(time.Time{})
			}
			_, wErr := conn.Write(buf[:nR])

			if nR < 512 || wErr != nil {
				_ = node.Close()
				requestServed = true
				break
			}
		}

		h.updateNodeStat(time.Since(startTime).Milliseconds(), currentNode, recalculationsCountState)

		if !requestServed && !responseReadStarted {
			_, _ = conn.Write([]byte("HTTP/1.1 504 Gateway Timeout\r\n\r\n"))
		}
		_ = conn.Close()

	}
}

//ServeHTTP is handler for standard http server
func (h *balancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	currentNode, recalculationsCountState := h.getCurrentNode()
	requestServed := false

	client := http.Client{}
	client.Timeout = time.Duration(h.cfg.nodeTimeout) * time.Millisecond

	if h.cfg.Nodes[currentNode].IsTLS {
		r.URL.Scheme = "https"
	} else {
		r.URL.Scheme = "http"
	}
	r.URL.Host = h.cfg.Nodes[currentNode].IpAndPort
	r.RequestURI = ""
	startTime := time.Now()
	resp, err := client.Do(r)
	responseTime := time.Since(startTime).Milliseconds()

	if err == nil {
		requestServed = true
	}

	h.updateNodeStat(responseTime, currentNode, recalculationsCountState)

	if !requestServed {
		w.WriteHeader(http.StatusGatewayTimeout)
	} else {
		for key, value := range resp.Header {
			for _, value := range value {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		buf := make([]byte, 512)
		for {
			n, err := resp.Body.Read(buf)
			_, _ = w.Write(buf[:n])
			if err != nil {
				break
			}

		}
	}
}

//getCurrentNode return current node and recalculationsCount
func (h *balancer) getCurrentNode() (int32, int32) {
	if h.d.requestsCount.get() >= h.cfg.nodeSelectionPeriod {
		chooseNewNode(h.d, h.d.recalculationsCount.get())
		if h.d.statsFreshnessCounter.add(1) >= h.cfg.statsCleaningPeriod {
			var i int32
			for i = 0; i < h.d.nodesAmount; i++ {
				h.d.nodes[i].overallResponseTime.set(0)
				h.d.nodes[i].requestsCount.set(1)
			}
			h.d.statsFreshnessCounter.set(0)
		}
		h.d.requestsCount.set(0)
	}
	h.d.requestsCount.add(1)
	currentNode := h.d.currentNode.get()
	return currentNode, h.d.recalculationsCount.get()
}

//updateNodeStat adds response time and requests count
func (h *balancer) updateNodeStat(responseTime int64, currentNode int32, recalculationsCountState int32) {
	//Make sure that stats will be added to node that was used at the start of function
	if recalculationsCountState == h.d.recalculationsCount.get() {
		h.d.nodes[currentNode].requestsCount.add(1)
		h.d.nodes[currentNode].overallResponseTime.add(responseTime)
	}
}
