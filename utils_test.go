package tftp

import (
	"bytes"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/jordwest/mock-conn"
)

// PacketEnd and TestPacketConn are mock PacketConn's,
// based on mock-conn for streaming connections,
// which uses a pair of pipes to simulate network connections
type PacketEnd struct {
	UnderlyingEnd mock_conn.End
}

type TestPacketConn struct {
	Client PacketEnd
	Server PacketEnd
}

func NewPacketConn() (result TestPacketConn) {
	i := mock_conn.NewConn()
	result.Client = PacketEnd{*i.Client}
	result.Server = PacketEnd{*i.Server}
	return
}

func (end *PacketEnd) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, err = end.UnderlyingEnd.Read(p)
	if err == nil && n == len(p) {
		// technically, we might have read right up to the end,
		// but I'm not sure we can detect that here...
		err = errors.New("didn't read to end of datagram")
	}
	addr = &net.UDPAddr{Port: 69}
	return
}

func (end *PacketEnd) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return end.UnderlyingEnd.Write(p)
}

// These functions are not expected to be called
func (end *PacketEnd) Close() error {
	return errors.New("Not Implemented")
}

func (end *PacketEnd) LocalAddr() net.Addr {
	return &net.UDPAddr{Port: 69}
}

func (end *PacketEnd) SetDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func (end *PacketEnd) SetReadDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func (end *PacketEnd) SetWriteDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

// build an array of test data packets
func generateTestData(numPackets int, endPacketLen int) (value [][]byte) {
	value = make([][]byte, numPackets)
	var cur byte
	for i := 0; i < numPackets-1; i++ {
		value[i] = make([]byte, 512)
		for j := 0; j < 512; j++ {
			value[i][j] = cur
			cur++
		}
	}
	// generate last packet, which could be empty, but must be less than 512 bytes
	value[numPackets-1] = make([]byte, endPacketLen)
	for k := 0; k < endPacketLen; k++ {
		value[numPackets-1][k] = cur
		cur++
	}
	return
}

func TestSendData(t *testing.T) {
	value := generateTestData(2, 2)
	conn := NewPacketConn()
	go sendData(&conn.Server, value, 10*time.Second, nil)
	buf := make([]byte, 517)
	n, _, error := conn.Client.ReadFrom(buf)
	if error != nil {
		t.Error(error)
	}
	packet := PacketData{}
	error = packet.Parse(buf[:n])
	if error != nil {
		t.Error(error)
	}

	if !bytes.Equal(value[0], packet.Data) {
		t.Error("First Data packet corrupt.")
	}

	ack := PacketAck{BlockNum: packet.BlockNum}
	_, error = conn.Client.WriteTo(ack.Serialize(), nil)
	if error != nil {
		t.Error(error)
	}

	n, _, error = conn.Client.ReadFrom(buf)
	if error != nil {
		t.Error(error)
	}
	packet = PacketData{}
	error = packet.Parse(buf[:n])
	if error != nil {
		t.Error(error)
	}

	if !bytes.Equal(value[1], packet.Data) {
		t.Error("Second Data packet corrupt.")
	}
}

func ReadAckPacket(t *testing.T, conn net.PacketConn) PacketAck {
	buf := make([]byte, 5)
	n, _, error := conn.ReadFrom(buf)
	if error != nil {
		t.Error(error)
	}
	resultPacket := PacketAck{}
	error = resultPacket.Parse(buf[:n])
	if error != nil {
		t.Error(error)
	}
	return resultPacket
}

func TestReceiveData(t *testing.T) {
	value := generateTestData(2, 2)
	conn := NewPacketConn()
	received := make(chan [][]byte)
	go func() {
		received <- receiveData(&conn.Server, 10*time.Second, nil)
	}()

	resultPacket := ReadAckPacket(t, &conn.Client)

	if resultPacket.BlockNum != 0 {
		t.Error("First Ack packet corrupt.")
	}
	p1 := PacketData{BlockNum: 1, Data: value[0]}
	_, error := conn.Client.WriteTo(p1.Serialize(), nil)

	if error != nil {
		t.Error(error)
	}

	resultPacket = ReadAckPacket(t, &conn.Client)
	if resultPacket.BlockNum != 1 {
		t.Error("Second Ack packet corrupt.")
	}

	p2 := PacketData{BlockNum: 2, Data: value[1]}
	_, error = conn.Client.WriteTo(p2.Serialize(), nil)

	if error != nil {
		t.Error(error)
	}

	resultPacket = ReadAckPacket(t, &conn.Client)
	if resultPacket.BlockNum != 2 {
		t.Error("Third Ack packet corrupt.")
	}

	resultData := <-received
	for i := range resultData {
		if !bytes.Equal(resultData[i], value[i]) {
			t.Error("Data corruption detected.")
		}
	}
}

type FailOnReadConn struct{}

func (e *FailOnReadConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	err = errors.New("Fake connection failure")
	return
}

func (e *FailOnReadConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	// no-op
	n = len(p)
	return
}

func (e *FailOnReadConn) Close() error {
	return errors.New("Not Implemented")
}

func (e *FailOnReadConn) LocalAddr() net.Addr {
	return &net.UDPAddr{Port: 69}
}

func (e *FailOnReadConn) SetDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func (e *FailOnReadConn) SetReadDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func (e *FailOnReadConn) SetWriteDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func TestBadSocketRead(t *testing.T) {

	conn := FailOnReadConn{}
	success := func(Packet) bool {
		return true
	}
	requestPacket := PacketAck{BlockNum: 7}
	resultPacket, err := sendAndWait(&conn, &requestPacket, time.Second, success, &net.UDPAddr{})
	if resultPacket != nil || err == nil {
		t.Error("Socket read failure did not result in an error")
	}
}

type FailOnWriteConn struct{}

func (e *FailOnWriteConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	// no-op
	n = len(p)
	addr = &net.UDPAddr{}
	return
}

func (e *FailOnWriteConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	err = errors.New("Fake connection failure")
	return
}

func (e *FailOnWriteConn) Close() error {
	return errors.New("Not Implemented")
}

func (e *FailOnWriteConn) LocalAddr() net.Addr {
	return &net.UDPAddr{Port: 69}
}

func (e *FailOnWriteConn) SetDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func (e *FailOnWriteConn) SetReadDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func (e *FailOnWriteConn) SetWriteDeadline(t time.Time) error {
	return errors.New("Not Implemented")
}

func TestBadSocketWrite(t *testing.T) {
	conn := FailOnWriteConn{}
	success := func(Packet) bool {
		return true
	}
	requestPacket := PacketAck{BlockNum: 7}
	resultPacket, err := sendAndWait(&conn, &requestPacket, time.Second, success, &net.UDPAddr{})
	if resultPacket != nil || err == nil {
		t.Error("Socket write failure did not result in an error")
	}
}

func TestMalformedRead(t *testing.T) {
	conn := NewPacketConn()
	success := func(Packet) bool {
		return true
	}
	requestPacket := PacketAck{BlockNum: 7}
	control := make(chan bool)
	go func() {
		resultPacket, err := sendAndWait(&conn.Server, &requestPacket, time.Second, success, &net.UDPAddr{})
		if resultPacket != nil || err == nil {
			t.Error("Socket read malformation did not result in an error")
		}
		control <- true
	}()
	buf := make([]byte, 517)
	conn.Client.ReadFrom(buf)
	conn.Client.WriteTo([]byte{1, 2, 3, 4}, &net.UDPAddr{})
	// don't exit until the goroutine is done
	<-control

}

func TestTimeOut(t *testing.T) {
	conn := NewPacketConn()
	success := func(Packet) bool {
		return true
	}
	requestPacket := PacketAck{BlockNum: 7}
	go sendAndWait(&conn.Server, &requestPacket, time.Second, success, &net.UDPAddr{})
	buf := make([]byte, 517)
	conn.Client.ReadFrom(buf)
	time.Sleep(2 * time.Second)
	buf2 := make([]byte, 517)
	n, _, _ := conn.Client.ReadFrom(buf2)
	if n < 1 {
		t.Error("Failed to resent toSend Packet")
	}

}

// testTimeOutSocket // hard
// testOpLog // hard-ish
