package tftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var store = MapDataStore{mapStore: make(map[string][][]byte)}

func handleConn(conn io.ReadWriter) {
	// TODO: handle setting TID/port
	// TODO: implement recover() here
	b, error := ioutil.ReadAll(conn)
	if error != nil {
		fmt.Fprintf(os.Stderr, "FATAL: An unknown error occurred - %s", error)
		return
	}
	request := PacketRequest{}
	error = request.Parse(b)
	if error != nil {
		sendError(conn, 0, error.Error())
	}
	if !strings.EqualFold(request.Mode, "octet") {
		sendError(conn, 0, "Only octet mode is supported") //unsupported mode
		return
	}
	switch request.Op {
	case OpRRQ:
		handleRead(conn, request)
	case OpWRQ:
		handleWrite(conn, request)
	}
}

func handleRead(conn io.ReadWriter, p PacketRequest) {
	if !store.keyExists(p.Filename) {
		sendError(conn, 1, fmt.Sprintf("File %s not found", p.Filename))
		return
	}
	sendData(conn, store.getData(p.Filename), time.Duration(time.Second*10))
}

func handleWrite(conn io.ReadWriter, p PacketRequest) {
	timeout := 10 * time.Second
	payload := receiveData(conn, timeout)
	if payload != nil {
		store.setData(p.Filename, payload)
	}
}
