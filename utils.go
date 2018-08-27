package tftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

// the largest payload in one data message is 512 bytes
const maxPayload int = 512

func handleConn(conn io.ReadWriter) {
	// TODO: handle setting TID/port
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

func sendError(conn io.Writer, code uint16, message string) {
	p := PacketError{Code: code, Msg: message}
	conn.Write(p.Serialize())
}

func handleRead(conn io.ReadWriter, p PacketRequest) {
	// check that key exists
	if !keyExists(p.Filename) {
		sendError(conn, 1, fmt.Sprintf("File %s not found", p.Filename))
		return
	}
	sendData(conn, getData(p.Filename), time.Duration(time.Second*10))
}

func sendData(conn io.ReadWriter, data [][]byte, timeout time.Duration) {
	// by iterating with int (32 bits or greater), we can handle
	// files up to at least 2^(31+9) bytes (1TB) in size, because
	// the cast to uint16 for the block number will roll over.
	for i, block := range data {
		// alternate sending data and awaiting acks
		// we will retry sending the data packet if
		// no ack after timeout
		for {
			dataPacket := PacketData{BlockNum: uint16(i + 1), Data: block[:]}
			conn.Write(dataPacket.Serialize())

			// timeout after a while
			ap := PacketAck{}
			c1 := make(chan uint16)
			cerror := make(chan error)
			go func() {
				for ap.BlockNum != uint16(i) {
					// ignore acks for other blocks, as they could
					// be duplicates, causing the Sorcerer's Apprentice
					// Syndrome (https://en.wikipedia.org/wiki/Sorcerer%27s_Apprentice_Syndrome)
					bytes, error := ioutil.ReadAll(conn)
					if error != nil {
						sendError(conn, 0, error.Error())
						cerror <- error
						break
					}
					error = ap.Parse(bytes)
					if error != nil {
						sendError(conn, 0, error.Error())
						cerror <- error
						break
					}
					if ap.BlockNum == uint16(i+1) {
						c1 <- ap.BlockNum
						break
					}
				}
			}()

			select {
			case <-c1:
				// ack received, move on
				break
			case <-cerror:
				// something went wrong, exit
				return
			case <-time.After(timeout):
				//retry, no-op
			}
		}
	}
}

func handleWrite(conn io.ReadWriter, p PacketRequest) {
	payload := receiveData(conn)
	if payload != nil {
		setData(p.Filename, payload)
	}
}

func receiveData(conn io.ReadWriter) [][]byte {
	dp := PacketData{}
	var payload = make([][]byte, 1)
	for {
		bytes, error := ioutil.ReadAll(conn)
		if error != nil {
			sendError(conn, 0, error.Error())
			return nil
		}
		error = dp.Parse(bytes)
		if error != nil {
			sendError(conn, 0, error.Error())
			return nil
		}
		// put the bytes somewhere
		payload = append(payload, dp.Data)
		sendAck(conn, dp.BlockNum)
		// any payload shorter than 512 bytes is a signal for EOF
		if len(dp.Data) < maxPayload {
			break
		}
	}
	return payload
}

func sendAck(conn io.Writer, blockNum uint16) {
	p := PacketAck{BlockNum: blockNum}
	conn.Write(p.Serialize())
}

var dataStore = make(map[string][][]byte)

// using a single RWMutex will lock the entire
// map on write.
// https://github.com/orcaman/concurrent-map
// may be a performance improvement.
var lock = sync.RWMutex{}

func keyExists(key string) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, ok := dataStore[key]
	return ok
}

// TODO: move this to interface for dependency injection
func getData(key string) [][]byte {
	// here we need a thread-safe map of string to 2d
	// array of bytes, whose shape is n x 512
	lock.RLock()
	defer lock.RUnlock()
	return dataStore[key]
}

func setData(key string, value [][]byte) {
	lock.Lock()
	defer lock.Unlock()
	dataStore[key] = value
}
