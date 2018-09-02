package tftp

import (
	"log"
	"net"
	"time"
)

// functions in this file are likely to be of value to clients
// as well as to the server, so they are kept separate.

// MaxPacketSize is the number of bytes read off the socket.
// The largest payload in one data message is 516 bytes, but read more to be sure
var MaxPacketSize = 2048

// the largest data block in one message is 512 bytes
const maxPayload int = 512

func sendData(conn net.PacketConn, data [][]byte, timeout time.Duration, dest net.Addr) {
	// by iterating with int (32 bits or greater), we can handle
	// files up to at least 2^(31+9) bytes (1TB) in size, because
	// the cast to uint16 for the block number will roll over.
	for i, block := range data {
		// alternate sending data and awaiting acks
		// we will retry sending the data packet if
		// no ack after timeout

		blockNum := uint16(i + 1)

		// ignore acks for other blocks, as they could
		// be duplicates, causing the Sorcerer's Apprentice
		// Syndrome (https://en.wikipedia.org/wiki/Sorcerer%27s_Apprentice_Syndrome)
		success := func(p Packet) (result bool) {
			v, ok := p.(*PacketAck)
			result = ok && v.BlockNum == blockNum
			if result {
				log.Printf("Received response packet: %+v\n", v)
			} else {
				log.Printf("Expected Ack %d, but got %+v\n", blockNum, p)
			}
			return
		}
		_, err := sendAndWait(conn, &PacketData{BlockNum: blockNum, Data: block[:]},
			timeout,
			success,
			dest)
		if err != nil {
			// TODO: check that this is the right error handling mechanism
			log.Printf("Failed to send data: %s", err.Error())
			return
		}
	}
}

func receiveData(conn net.PacketConn, timeout time.Duration, dest net.Addr) [][]byte {
	var dp *PacketData
	ack := PacketAck{BlockNum: 0}
	var payload = make([][]byte, 1)
	// any payload shorter than 512 bytes is a signal for EOF
	for dp == nil || len(dp.Data) == maxPayload {
		success := func(p Packet) (result bool) {
			dataPacket, ok := p.(*PacketData)
			blockNum := ack.BlockNum + 1
			result = ok && dataPacket.BlockNum == blockNum
			if result {
				log.Printf("Received response packet: %+v\n", dataPacket)
			} else {
				log.Printf("Expected Data Block %d, but got %+v\n", blockNum, p)
			}
			return
		}
		packet, err := sendAndWait(conn, &ack, timeout, success, dest)
		if err != nil {
			log.Printf("Failed to receive data: %s", err.Error())
			return nil
		}
		// cast will always succeed, because we've already cast in success criteria
		dp, _ = packet.(*PacketData)

		// put the bytes somewhere
		payload = append(payload, dp.Data)
		ack.BlockNum++
	}
	conn.WriteTo(ack.Serialize(), dest)
	return payload
}

type SuccessCriteria func(Packet) bool

func sendAndWait(conn net.PacketConn, toSend Packet, timeout time.Duration, success SuccessCriteria, dest net.Addr) (responsePacket Packet, err error) {
	cerror := make(chan error)
	cresult := make(chan Packet)
	for {
		log.Printf("Sending response: %+v\n", toSend)
		_, err = conn.WriteTo(toSend.Serialize(), dest)
		if err != nil {
			// if we fail to write, we should exit, as we can't send an error packet
			return
		}

		go func() {
			var received Packet
			// until we have success, do this
			for received == nil || !success(received) {
				bytes := make([]byte, MaxPacketSize)
				n, _, error := conn.ReadFrom(bytes)
				if error != nil {
					sendError(conn, 0, error.Error(), dest)
					cerror <- error
					return
				}
				// trim any trailing bytes
				bytes = bytes[:n]
				received, error = ParsePacket(bytes)
				if error != nil {
					log.Printf("Received garbage data, still waiting for packet.")
					sendError(conn, 0, error.Error(), dest)
					cerror <- error
					return
				}
			}
			// if we've exited the loop, success is true
			cresult <- received
		}()

		select {
		case err = <-cerror:
			// do we really want to return the error here, or keep trying?
			return
		case responsePacket = <-cresult:
			return
		case <-time.After(timeout):
			// resend
			continue
		}
	}
}

func sendError(conn net.PacketConn, code uint16, message string, dest net.Addr) {
	// sendError should never block
	go func() {
		p := PacketError{Code: code, Msg: message}
		conn.WriteTo(p.Serialize(), dest)
	}()
}

var OpLogger = log.Logger{}

func logPacket(b []byte, op string) {
	p, err := ParsePacket(b)
	if err == nil {
		OpLogger.Printf("%s packet: %+v", op, p)
	} else {
		OpLogger.Printf("%s garbage data: %+v", op, b)
	}

}

// PacketConnLogger wraps a packet connection with log statements
// allowing us to trace what was written.  For the sake of this project
// the buffers are assumed to contain TFTP packets
type PacketConnLogger struct {
	PacketConn net.PacketConn
}

func (conn *PacketConnLogger) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, addr, err = conn.PacketConn.ReadFrom(p)
	logPacket(p[:n], "Read")
	return
}

func (conn *PacketConnLogger) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	logPacket(p, "Wrote")
	return conn.PacketConn.WriteTo(p, addr)
}

// These functions are not expected to be called
func (conn *PacketConnLogger) Close() error {
	return conn.PacketConn.Close()
}

func (conn *PacketConnLogger) LocalAddr() net.Addr {
	return conn.PacketConn.LocalAddr()
}

func (conn *PacketConnLogger) SetDeadline(t time.Time) error {
	return conn.PacketConn.SetDeadline(t)
}

func (conn *PacketConnLogger) SetReadDeadline(t time.Time) error {
	return conn.PacketConn.SetReadDeadline(t)
}

func (conn *PacketConnLogger) SetWriteDeadline(t time.Time) error {
	return conn.PacketConn.SetWriteDeadline(t)
}
