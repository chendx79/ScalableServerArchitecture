package service

import "time"

const (
	PING_PORT_NUMBER = 9999
	PING_INTERVAL    = 1000 * time.Millisecond //  Once per second
	PEER_EXPIRY      = 5000 * time.Millisecond //  Five seconds and it's gone
)

const (
	CHAT    = "CHAT"
	SESSION = "SESSION"
)
