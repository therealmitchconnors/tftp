package tftp

import (
	"io/ioutil"
	"io"
	"ioutil"
)

// the largest payload in one data message is 512 bytes
const maxPayload := 512

func handleConn(conn io.ReadWriter) {
	b, error := ioutil.ReadAll(conn)
	// todo: error handling
	request := PacketRequest{}
	request.Parse(b)
	switch request.Op {
	case OpRRQ:
		handleRead(conn, request)
	case OpWRQ:
		handleWrite(conn, request)
	default:
		sendError(conn)
	}
}

func sendError(conn io.Writer) {
	// TODO: Add params for error
	p := PacketError{}
	conn.Write(p.Serialize())
}

func handleRead(conn io.ReadWriter, p PacketRequest) {
	// check that key exists
	// TODO: refactor to handleOutgoingData for client use
	i := 0
	for {
		// alternate sending data and awaiting acks
		remainingBytes := sendDataBlock(conn, p.FileName, i)
		ap := PacketAck{}
		ap.Parse(ioutil.ReadAll(conn))
		// TODO: check for error here
		if remainingBytes <= 0 {
			// exit when no more data or error
			break
		}
		i++
	}
}

func handleWrite(conn io.ReadWriter, p PacketRequest) {
	dp := PacketData{}
	// TODO: refactor this to handleIncomingData for client use
	for {
		// TODO: handle errors
		dp.Parse(ioutil.ReadAll(conn))
		// put the bytes somewhere
		sendAck(conn, dp.BlockNum)
		// any payload shorter than 512 bytes is a signal for EOF
		if len(dp.Data) < maxPayload {
			break
		}
	}
	// finalize the bytes
}

// golang's math.Min casts to float, which is unnecessary
func Min(a, b int) int {
	a > b ? return b : return a
}

func sendDataBlock(conn io.Writer, key string, blockNum int) int {
	data := getData(key)
	sliceStart := blockNum * maxPayload
	sliceEnd := Min(len(data), sliceStart + maxPayload)
	p := PacketData{BlockNum: blockNum, Data: data[sliceStart:sliceEnd]}
	conn.Write(p.Serialize())
	return len(data) - sliceEnd
}

func sendAck(conn io.Writer, blockNume int) {
	p := PacketAck{BlockNum: blockNum}
	conn.Write(p.Serialize())
}

// TODO: move this to interface for dependency injection
func getData(key string,) []byte {

}