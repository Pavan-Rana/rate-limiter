package limiter

import "time"

type Policy struct {
	Limit	 	int
	Window	 	time.Duration
	BurstLimit 	int
}

var DefaultPolicy = Policy{
	Limit: 	100,
	Window: time.Minute,
}