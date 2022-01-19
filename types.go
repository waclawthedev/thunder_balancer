package main

import "sync"

//Node contains data and stats for node
type Node struct {
	ipAndPort           string
	requestsCount       AtomicInt64 //requests count performed
	overallResponseTime AtomicInt64 //response time sum (to calculate avg response time)
	isTLS               bool
}

//DataSet contains general data for balancer runtime
type DataSet struct {
	nodes                 []Node //array with nodes (url)
	nodesAmount           int32
	currentNode           AtomicInt32 //index in arrays of less loaded node
	chooseNewNodeMutex    sync.Mutex  //to ensure that chooseNewNode will be executed synchronously
	requestsCount         AtomicInt64 //requests count across system after last chooseNewNode
	recalculationsCount   AtomicInt32 //to track the changes of currentNode across requests
	statsFreshnessCounter AtomicInt64 //to track lifetime of stats
}

//newDataSetFromConfig returns initialized empty DataSet
func newDataSetFromConfig(cfg Config) (d *DataSet) {
	d = &DataSet{
		currentNode:           AtomicInt32{new(int32)},
		chooseNewNodeMutex:    sync.Mutex{},
		statsFreshnessCounter: AtomicInt64{new(int64)},
		requestsCount:         AtomicInt64{new(int64)},
		recalculationsCount:   AtomicInt32{new(int32)},
	}
	for i, nodeFromEnv := range cfg.Nodes {
		d.nodes = append(d.nodes, Node{
			ipAndPort:           nodeFromEnv.IpAndPort,
			overallResponseTime: AtomicInt64{new(int64)},
			requestsCount:       AtomicInt64{new(int64)},
			isTLS:               nodeFromEnv.IsTLS,
		})
		//if nodeFromEnv.IsTLS
		d.nodes[i].overallResponseTime.set(1)
		d.nodes[i].overallResponseTime.set(1)
		d.nodes[i].requestsCount.set(1)

	}
	d.nodesAmount = int32(len(d.nodes))
	d.currentNode.set(0)
	d.statsFreshnessCounter.set(0)
	d.requestsCount.set(0)
	return
}
