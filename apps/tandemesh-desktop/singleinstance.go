package main

import (
	"log"
	"net"
)

// acquireSingleInstance keeps a fixed loopback port open for the lifetime of
// the process. A second desktop instance fails to bind and exits.
func acquireSingleInstance() (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:47831")
	if err != nil {
		log.Printf("another Tandemesh Desktop instance is already running")
		return nil, err
	}
	return listener, nil
}
