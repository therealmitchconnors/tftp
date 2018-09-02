package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/therealmitchconnors/tftp"
)

type uInt16Value struct {
	val uint16
}

func (v *uInt16Value) String() string {
	return string(v.val)
}

func (v *uInt16Value) Set(s string) error {
	if u, err := strconv.ParseUint(s, 10, 16); err != nil {
		return err
	} else {
		v.val = uint16(u)
		return nil
	}
}

func main() {
	// port number defaults to 69
	portFlag := uInt16Value{69}
	flag.Var(&portFlag, "port", "The port tftpd will listen on")

	// maxPacketSize defaults to 2048
	maxPacketSizeFlag := uInt16Value{uint16(tftp.MaxPacketSize)}
	flag.Var(&maxPacketSizeFlag, "max-packet-size", "The max transmission unit for UDP reads.  Larger packets will truncate, smaller values are more efficient.")

	opLogFile := flag.String("oplog", "./operations.log", "The destination for operation logs")

	flag.Parse()

	tftp.MaxPacketSize = int(maxPacketSizeFlag.val)
	f, err := os.OpenFile(*opLogFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	tftp.OpLogger.SetOutput(f)
	log.Printf("tftpd is listening on port %d\n", portFlag.val)

	udpser, err := net.ListenUDP("udp", &net.UDPAddr{Port: int(portFlag.val)})

	// wrap our server in the operations logger
	ser := &tftp.PacketConnLogger{PacketConn: udpser}
	if err != nil {
		log.Fatal(err)
	}
	defer ser.Close()
	for {
		// Wait for a connection.
		buf := make([]byte, maxPacketSizeFlag.val)
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
