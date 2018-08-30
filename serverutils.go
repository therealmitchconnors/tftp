package tftp

import (
	"fmt"
	"net"
	"strings"
	"time"
)

var store = MapDataStore{mapStore: make(map[string][][]byte)}

func handleReq(buf []byte, addr net.UDPAddr) {
	// TODO: implement recover() here
	request := PacketRequest{}
	error := request.Parse(buf)

	// negotiate new connection using TID
	conn, error := net.ListenUDP("udp", nil)
	// I don't love passing the client addr to every funciton,
	// but it allows us to use only the PacketConn interface
	// which is better
	if error != nil {
		sendError(conn, 0, error.Error(), &addr)
	}
	if !strings.EqualFold(request.Mode, "octet") {
		sendError(conn, 0, "Only octet mode is supported", &addr) //unsupported mode
		return
	}

	switch request.Op {
	case OpRRQ:
		handleRead(conn, request, &addr)
	case OpWRQ:
		handleWrite(conn, request, &addr)
	}
}

func handleRead(conn net.PacketConn, p PacketRequest, addr net.Addr) {
	if !store.keyExists(p.Filename) {
		sendError(conn, 1, fmt.Sprintf("File %s not found", p.Filename), addr)
		return
	}
	sendData(conn, store.getData(p.Filename), time.Duration(time.Second*10), addr)
}

func handleWrite(conn net.PacketConn, p PacketRequest, addr net.Addr) {
	timeout := 10 * time.Second
	payload := receiveData(conn, timeout, addr)
	if payload != nil {
		store.setData(p.Filename, payload)
	}
}
