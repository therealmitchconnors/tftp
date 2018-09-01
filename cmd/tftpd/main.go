package main

import (
	"log"
	"net"

	"igneous.io/tftp"
)

func main() {
	// TODO: get port number from cmd line
	port := 69
	ser, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
	if err != nil {
		log.Fatal(err)
	}
	defer ser.Close()
	for {
		// Wait for a connection.
		// TODO: make mtu default to 516 with cl param
		// 516 is the longest possible tftp packet (assuming long file names aren't allowed)
		buf := make([]byte, 516)
		_, addr, err := ser.ReadFrom(buf)
		if err != nil {
			log.Fatal(err)
		}
		// Handle the request in go routine, allowing
		// the main thread to keep accepting new connections.

		// Stale clients will cause stale go routines,
		// but we can handle millions of go routines in an app,
		// so this is likely a tolerable trade-off
		go func() {
			tftp.HandleReq(buf, *addr.(*net.UDPAddr))
		}()
	}
}
