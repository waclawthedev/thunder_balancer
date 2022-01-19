package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
)

//Config - to store data for work
type Config struct {
	Nodes []struct {
		IpAndPort string `json:"node"`
		IsTLS     bool   `json:"is_tls"`
	}
	nodeTimeout         int
	hostAndPort         string
	engine              string
	tlsUsed             bool
	tlsCert             string
	tlsKey              string
	nodeSelectionPeriod int64
	statsCleaningPeriod int64
}

//newConfigFromEnv reads and returns Config
func newConfigFromEnv() Config {
	var cfg Config
	err := json.Unmarshal([]byte(os.Getenv("NODES")), &cfg.Nodes)
	if err != nil {
		log.Fatalf("check the NODES env variable - can't parse current input")
	}

	cfg.nodeSelectionPeriod, err = strconv.ParseInt(os.Getenv("SELECT_NODE_PERIOD"), 10, 64)
	if err != nil {
		log.Fatalf("check the SELECT_NODE_PERIOD env variable - can't parse as number")
	}
	cfg.statsCleaningPeriod, err = strconv.ParseInt(os.Getenv("CLEAN_STATS_PERIOD"), 10, 64)
	if err != nil {
		log.Fatalf("check the CLEAN_STATS_PERIOD env variable - can't parse as number")
	}

	switch os.Getenv("TLS") {
	case "ON":
		cfg.tlsUsed = true
		cfg.tlsCert = os.Getenv("TLS_CERT")
		cfg.tlsKey = os.Getenv("TLS_KEY")
	case "OFF":
		cfg.tlsUsed = false
	default:
		log.Fatalf("define the TLS env variable (ON or OFF)")
	}

	cfg.engine = os.Getenv("ENGINE")

	if os.Getenv("HOST_PORT") == "" {
		log.Fatalf("define host and port to listen like. example: 0.0.0.0:8080")
	}
	cfg.hostAndPort = os.Getenv("HOST_PORT")
	cfg.nodeTimeout, err = strconv.Atoi(os.Getenv("NODE_TIMEOUT"))
	if err != nil {
		log.Fatalf("define NODE_TIMEOUT as number")
	}

	return cfg
}
