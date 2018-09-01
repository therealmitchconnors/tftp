package tftp

import (
	"fmt"
	"net"
	"strings"
	"time"
)

var store = MapDataStore{mapStore: make(map[string][][]byte)}

// ServerDependencies makes more sence as an interface, but
// interfaces cannot be anonymously implemented, while structs
// of functions can!
type ServerDependencies struct {
	openRandomSendPort func() (net.PacketConn, error)
	sendError          func(conn net.PacketConn, code uint16, message string, dest net.Addr)
	handleRead         func(conn net.PacketConn, p PacketRequest, addr net.Addr)
	handleWrite        func(conn net.PacketConn, p PacketRequest, addr net.Addr)
}

type UtilDependencies struct {
	sendData    func(conn net.PacketConn, data [][]byte, timeout time.Duration, dest net.Addr)
	receiveData func(conn net.PacketConn, timeout time.Duration, dest net.Addr) [][]byte
	sendError   func(conn net.PacketConn, code uint16, message string, dest net.Addr)
}

func HandleReq(buf []byte, addr net.UDPAddr) {
	// These objects just inject the functions for production use
	productionUtils := UtilDependencies{
		sendData: func(conn net.PacketConn, data [][]byte, timeout time.Duration, dest net.Addr) {
			sendData(conn, data, timeout, dest)
		},
		receiveData: func(conn net.PacketConn, timeout time.Duration, dest net.Addr) [][]byte {
			return receiveData(conn, timeout, dest)
		},
		sendError: func(conn net.PacketConn, code uint16, message string, dest net.Addr) {
			sendError(conn, code, message, dest)
		},
	}
	productionDependencies := ServerDependencies{
		openRandomSendPort: func() (net.PacketConn, error) {
			return net.ListenUDP("udp", nil)
		},
		sendError: func(conn net.PacketConn, code uint16, message string, dest net.Addr) {
			productionUtils.sendError(conn, code, message, dest)
		},
		handleRead: func(conn net.PacketConn, p PacketRequest, addr net.Addr) {
			handleRead(conn, p, addr, productionUtils)
		},
		handleWrite: func(conn net.PacketConn, p PacketRequest, addr net.Addr) {
			handleWrite(conn, p, addr, productionUtils)
		},
	}

	handleReqDep(buf, addr, productionDependencies)
}

func handleReqDep(buf []byte, addr net.UDPAddr, dep ServerDependencies) {
	// TODO: implement recover() here
	request := PacketRequest{}
	error := request.Parse(buf)

	// negotiate new connection using TID
	conn, error := dep.openRandomSendPort()
	// conn, error := net.ListenUDP("udp", nil)
	// I don't love passing the client addr to every funciton,
	// but it allows us to use only the PacketConn interface
	// which is better
	if error != nil {
		dep.sendError(conn, 0, error.Error(), &addr)
	}
	if !strings.EqualFold(request.Mode, "octet") {
		dep.sendError(conn, 0, "Only octet mode is supported", &addr) //unsupported mode
		return
	}

	switch request.Op {
	case OpRRQ:
		dep.handleRead(conn, request, &addr)
	case OpWRQ:
		dep.handleWrite(conn, request, &addr)
	}
}

func handleRead(conn net.PacketConn, p PacketRequest, addr net.Addr, dep UtilDependencies) {
	if !store.keyExists(p.Filename) {
		dep.sendError(conn, 1, fmt.Sprintf("File %s not found", p.Filename), addr)
		return
	}
	dep.sendData(conn, store.getData(p.Filename), time.Duration(time.Second*10), addr)
}

func handleWrite(conn net.PacketConn, p PacketRequest, addr net.Addr, dep UtilDependencies) {
	timeout := 10 * time.Second
	payload := dep.receiveData(conn, timeout, addr)
	if payload != nil {
		store.setData(p.Filename, payload)
	}
}
