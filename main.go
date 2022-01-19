package main

func main() {
	cfg := newConfigFromEnv()
	s := newServer(cfg)
	s.start()
}
