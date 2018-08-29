package tftp

import (
	"net"
	"time"
)

// functions in this file are likely to be of value to clients
// as well as to the server, so they are kept separate.

// the largest payload in one data message is 512 bytes
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
		success := func(p Packet) bool {
			v, ok := p.(*PacketAck)
			return ok && v.BlockNum == blockNum
		}
		_, err := sendAndWait(conn, &PacketData{BlockNum: blockNum, Data: block[:]},
			timeout,
			success,
			dest)
		if err != nil {
			// TODO: check that this is the right error handling mechanism
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
		success := func(p Packet) bool {
			dataPacket, ok := p.(*PacketData)
			return ok && dataPacket.BlockNum == ack.BlockNum+1
		}
		packet, err := sendAndWait(conn, &ack, timeout, success, dest)
		if err != nil {
			return nil
		}
		// cast will always succeed, because we've already cast in success criteria
		dp, _ := packet.(*PacketData)

		// put the bytes somewhere
		payload = append(payload, dp.Data)
	}
	return payload
}

type SuccessCriteria func(Packet) bool

func sendAndWait(conn net.PacketConn, toSend Packet, timeout time.Duration, success SuccessCriteria, dest net.Addr) (responsePacket Packet, err error) {
	cerror := make(chan error)
	cresult := make(chan Packet)
	for {
		conn.WriteTo(toSend.Serialize(), dest)
		// TODO: Error handle here

		go func() {
			var received Packet
			// until we have success, do this
			for received == nil || !success(received) {
				bytes := make([]byte, 516)
				n, _, error := conn.ReadFrom(bytes)
				if error != nil {
					sendError(conn, 0, error.Error(), dest)
					cerror <- error
					return
				}
				received, error = ParsePacket(bytes)
				if error != nil {
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
	p := PacketError{Code: code, Msg: message}
	conn.WriteTo(p.Serialize(), dest)
}
