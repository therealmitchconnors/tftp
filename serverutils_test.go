package tftp

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func setupTestInjections(conn net.PacketConn) (testUtils UtilDependencies, testServerUtils ServerDependencies, callCounter map[string][]map[string]interface{}) {
	// These objects just inject the functions for production use
	callCounter = make(map[string][]map[string]interface{})
	countCall := func(callName string, params map[string]interface{}) {
		callCounter[callName] = append(callCounter[callName], params)
	}
	testUtils = UtilDependencies{
		sendData: func(conn net.PacketConn, data [][]byte, timeout time.Duration, dest net.Addr) {
			countCall("sendData", map[string]interface{}{"conn": conn, "data": data, "timeout": timeout, "dest": dest})
		},
		receiveData: func(conn net.PacketConn, timeout time.Duration, dest net.Addr) [][]byte {
			countCall("receiveData", map[string]interface{}{"conn": conn, "timeout": timeout, "dest": dest})
			return make([][]byte, 1)
		},
		sendError: func(conn net.PacketConn, code uint16, message string, dest net.Addr) {
			countCall("sendError", map[string]interface{}{"conn": conn, "code": code, "message": message, "dest": dest})
		},
	}
	testServerUtils = ServerDependencies{
		openRandomSendPort: func() (net.PacketConn, error) {
			countCall("openRandomSendPort", map[string]interface{}{})
			return conn, nil
		},
		sendError: func(conn net.PacketConn, code uint16, message string, dest net.Addr) {
			// this will be counted in testUtils
			testUtils.sendError(conn, code, message, dest)
		},
		handleRead: func(conn net.PacketConn, p PacketRequest, addr net.Addr) {
			countCall("handleRead", map[string]interface{}{"conn": conn, "p": p, "addr": addr})
		},
		handleWrite: func(conn net.PacketConn, p PacketRequest, addr net.Addr) {
			countCall("handleWrite", map[string]interface{}{"conn": conn, "p": p, "addr": addr})
		},
	}
	return
}

func TestHandleReadReq(t *testing.T) {
	testPacketConn := NewPacketConn()
	_, testServerUtils, callCounter := setupTestInjections(&testPacketConn.Server)

	fname := "foo.txt"
	p := PacketRequest{Op: OpRRQ, Mode: "octet", Filename: fname}

	handleReqDep(p.Serialize(), net.UDPAddr{}, testServerUtils)

	checkErrors(callCounter, t)

	// handleRead should have been called once
	calls, ok := callCounter["handleRead"]
	if !ok || len(calls) < 1 {
		t.Fatal("Read Request did not call handleRead()")
	}
	reqPacket := calls[0]["p"].(PacketRequest)
	if reqPacket.Filename != fname {
		t.Errorf("Expected filename %s, but got %s", fname, reqPacket.Filename)
	}
}

func checkErrors(callCounter map[string][]map[string]interface{}, t *testing.T) {
	calls, ok := callCounter["sendError"]
	if ok {
		errMsg := calls[0]["message"].(string)
		t.Errorf("Unexpected Error sent: %s", errMsg)
	}
}

func TestHandleWriteReq(t *testing.T) {
	testPacketConn := NewPacketConn()
	_, testServerUtils, callCounter := setupTestInjections(&testPacketConn.Server)

	fname := "foo.txt"
	p := PacketRequest{Op: OpWRQ, Mode: "octet", Filename: fname}

	handleReqDep(p.Serialize(), net.UDPAddr{}, testServerUtils)

	checkErrors(callCounter, t)

	// handleRead should have been called once
	calls, ok := callCounter["handleWrite"]
	if !ok || len(calls) < 1 {
		t.Fatal("Read Request did not call handleWrite()")
	}
	reqPacket := calls[0]["p"].(PacketRequest)
	if reqPacket.Filename != fname {
		t.Errorf("Expected filename %s, but got %s", fname, reqPacket.Filename)
	}
}

func TestHandleWrite(t *testing.T) {
	testPacketConn := NewPacketConn()
	testUtils, _, callCounter := setupTestInjections(&testPacketConn.Server)

	p := PacketRequest{Op: OpWRQ, Mode: "octet", Filename: "fname"}
	handleWrite(&testPacketConn.Server, p, &net.UDPAddr{}, testUtils)

	checkErrors(callCounter, t)

	if len(callCounter["receiveData"]) < 1 {
		t.Error("handleWrite failed to call receiveData")
	}

	inputData := testUtils.receiveData(&testPacketConn.Server, time.Second, &net.UDPAddr{})

	for i, v := range store.getData("fname") {
		if !bytes.Equal(v, inputData[i]) {
			t.Error("input data does not match stored data.")
		}
	}
}

func TestHandleRead(t *testing.T) {
	testPacketConn := NewPacketConn()
	testUtils, _, callCounter := setupTestInjections(&testPacketConn.Server)

	fname := "readfile"
	payload := []byte{42}
	payload2d := [][]byte{payload}

	store.setData(fname, payload2d)

	p := PacketRequest{Op: OpRRQ, Mode: "octet", Filename: fname}
	handleRead(&testPacketConn.Server, p, &net.UDPAddr{}, testUtils)

	checkErrors(callCounter, t)

	if len(callCounter["sendData"]) < 1 {
		t.Error("handleRead failed to call sendData")
	}

	for i, v := range store.getData(fname) {
		if !bytes.Equal(v, payload2d[i]) {
			t.Error("input data does not match stored data.")
		}
	}
}

func TestHandleReqBadMode(t *testing.T) {
	testPacketConn := NewPacketConn()
	_, testServerUtils, callCounter := setupTestInjections(&testPacketConn.Server)

	fname := "foo.txt"
	p := PacketRequest{Op: OpWRQ, Mode: "ascii", Filename: fname}

	// We expect an error packet in response, as ascii mode is not supported
	handleReqDep(p.Serialize(), net.UDPAddr{}, testServerUtils)

	_, ok := callCounter["sendError"]
	if !ok {
		t.Errorf("Unsupported ASCII Mode Accepted by server")
	}
}

func TestHandleReadMissingFile(t *testing.T) {

	testPacketConn := NewPacketConn()
	testUtils, _, callCounter := setupTestInjections(&testPacketConn.Server)

	fname := "readfile"

	// reset the store so tests don't bleed state here
	// TODO: remove global state
	store = MapDataStore{mapStore: make(map[string][][]byte)}
	// this is just like testHandleRead, but we don't set the file first
	// store.setData(fname, payload2d)

	p := PacketRequest{Op: OpRRQ, Mode: "octet", Filename: fname}
	handleRead(&testPacketConn.Server, p, &net.UDPAddr{}, testUtils)

	_, ok := callCounter["sendError"]
	if !ok {
		t.Errorf("Read for non-existing file didn't result in error")
	}
}

// noSuchKey
