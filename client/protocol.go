package client

import (
	"bufio"
	"context"
	"encoding/binary"
	"icetea/pkg"
)

func ReadMsg(ctx context.Context, reader *bufio.Reader) ([]byte, bool, error) {
	// reader.Reset(bytes.NewBuffer(leftRead))
	var (
		err    error
		header []byte
		// MQTT固定报文头最少有两个字节
		peekCount = 2
		gotHeader = false
	)
	for {
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		default:
			// MQTT 最多只允许使用四个字节表示剩余长度
			if peekCount > 5 {
				return nil, false, pkg.ErrMQTTCodeProtocolError
			}
			if header, err = reader.Peek(peekCount); err != nil {
				return header, false, err
			}
			if header[peekCount-1] >= 0x80 {
				peekCount++
			}
			gotHeader = true
		}
		if gotHeader {
			break
		}
	}
	var (
		remLen, m = binary.Uvarint(header[1:])
		remaining = 1 + int(remLen) + m
		recv      = make([]byte, remaining)
		offset    = len(recv) - remaining
	)

	for offset != remaining {
		select {
		case <-ctx.Done():
			return recv, false, ctx.Err()
		default:
			var n int
			n, err = reader.Read(recv[offset:])
			offset += n
			if err != nil {
				remaining -= offset
				return recv[0:offset], false, err
			}
		}
	}
	return recv, true, nil
}
