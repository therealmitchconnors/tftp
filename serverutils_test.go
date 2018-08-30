package tftp

import (
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
			countCall("handleRead", map[string]interface{}{"conn": conn, "p": p, "addr": addr})
		},
	}
	return
}

func TestHandleReq(t *testing.T) {
	testPacketConn := NewPacketConn()
	_, testServerUtils, callCounter := setupTestInjections(&testPacketConn.Server)

	fname := "foo.txt"
	p := PacketRequest{Op: OpRRQ, Mode: "fhqwgads", Filename: fname}

	handleReqDep(p.Serialize(), net.UDPAddr{}, testServerUtils)

	calls, ok := callCounter["sendError"]
	if ok {
		errMsg := calls[0]["message"].(string)
		t.Errorf("Unexpected Error sent: %s", errMsg)
	}

	// handleRead should have been called once
	calls, ok = callCounter["handleRead"]
	if !ok || len(calls) < 1 {
		t.Fatal("Read Request did not call handleRead()")
	}
	reqPacket := calls[0]["p"].(PacketRequest)
	if reqPacket.Filename != fname {
		t.Errorf("Expected filename %s, but got %s", fname, reqPacket.Filename)
	}
}
