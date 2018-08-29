package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	// TODO: get port number from cmd line
	port := 2000
	ser, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	defer ser.Close()
	for {
		// Wait for a connection.
		buf := make([]byte, 516) // 516 is the longest possible tftp packet (assuming long file names aren't allowed)
		n, addr, err := ser.ReadFrom(buf)
		if err != nil {
			log.Fatal(err)
		}
		// Handle the request in another thread, allowing
		// the main thread to keep accepting new connections.
		go func(n int, addr net.Addr, buf []byte) {
			handleIt(n, addr, buf)
		}(n, addr, buf)
	}
}

func handleIt(n int, addr net.Addr, buf []byte) {

}
