package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	// TODO: get port number from cmd line
	port := 2000
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// Handle the request in another thread, allowing
		// the main thread to keep accepting new connections.
		go func(c net.Conn) {
			c.Close()
		}(conn)
	}
}
