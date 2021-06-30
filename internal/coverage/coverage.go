package coverage

type Coverage struct {
	RealtimeProxies []struct{} `json:"realtime_proxies"`
	ZmqSocket       string     `json:"zmq_socket"`
	Key             string     `json:"key"`
}
