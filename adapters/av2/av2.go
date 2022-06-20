package av2

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"time"
)

type Command byte
type Response byte

const (
	COMMAND_ON           Command = 0x21
	COMMAND_STANDBY      Command = 0x22
	COMMAND_MUTE_ON      Command = 0x24
	COMMAND_MUTE_OFF     Command = 0x25
	COMMAND_INPUT_VIP1   Command = 0x2f
	COMMAND_INPUT_VIP2   Command = 0x30
	COMMAND_INPUT_OP1    Command = 0x35
	COMMAND_INPUT_OP2    Command = 0x36
	COMMAND_STATUS_QUERY Command = 0x69
	COMMAND_VOLUME       Command = 0x23 // requires level 0-99 as second byte
)

const RESPONSE_SYSTEM_STATUS Response = 0x69

type SystemStatusResponse struct {
	status [4]byte
}

type UnknownResponse struct {
	MsgCode Response
	Data    []byte
}

type Input uint8

const INPUT_UNKNOWN Input = 0x0
const INPUT_VIP1 Input = 0x1
const INPUT_VIP2 Input = 0x2
const INPUT_OP1 Input = 0x7
const INPUT_OP2 Input = 0x8

type responseConstraint interface {
	UnknownResponse | SystemStatusResponse
}

type callback[T responseConstraint] func(T) error

type NaimAV2 struct {
	writer io.Writer // must not be buffered, as Naim protocol requires specific timing
	reader *bufio.Reader

	systemStatusCallbacks    []callback[SystemStatusResponse]
	unknownResponseCallbacks []callback[UnknownResponse]

	lastSend time.Time
}

func NewNaim(reader io.Reader, writer io.Writer) NaimAV2 {
	bufReader := bufio.NewReader(reader)
	return NaimAV2{
		reader:                   bufReader,
		writer:                   writer,
		systemStatusCallbacks:    []callback[SystemStatusResponse]{},
		unknownResponseCallbacks: []callback[UnknownResponse]{},
	}
}

func (n *NaimAV2) AddSystemStatusHandler(fn callback[SystemStatusResponse]) {
	n.systemStatusCallbacks = append(n.systemStatusCallbacks, fn)
}

func (n *NaimAV2) AddUnknownResponseHandler(fn callback[UnknownResponse]) {
	n.unknownResponseCallbacks = append(n.unknownResponseCallbacks, fn)
}

type response interface {
	callHandlers(n *NaimAV2) error
}

func (n *NaimAV2) Read() error {
	// "#AV2 " + <1-28 bytes> + 0xFF
	var start = []byte("#AV2 ")
	const minLength = len("#AV2 bb")
	buf, err := n.reader.ReadBytes(0xFF)
	if err != nil {
		if err == io.EOF && len(buf) == 0 {
			return err
		}
		return fmt.Errorf("unable to read complete naim message: %w", err)
	}
	if len(buf) < minLength {
		return fmt.Errorf("message below minimum length")
	}
	if bytes.Compare(start, buf[:len(start)]) != 0 {
		return fmt.Errorf("unexpected message start: '%v'", buf[:len(start)])
	}
	// strip off the starting char, device ID, and the delimiter
	msgCode := Response(buf[len(start)])
	buf = buf[len(start)+1 : len(buf)-1]
	var r response
	switch msgCode {
	case RESPONSE_SYSTEM_STATUS:
		r, err = n.newSystemStatusResponse(buf)
	default:
		r = UnknownResponse{
			MsgCode: msgCode,
			Data:    buf,
		}
	}
	if err != nil {
		return fmt.Errorf("unable to parse data for message type %v: %w", msgCode, err)
	}
	if err = r.callHandlers(n); err != nil {
		return fmt.Errorf("unable to call handlers for message type %v: %w", msgCode, err)
	}
	return nil
}

func (n *NaimAV2) ReadAll() error {
	for {
		if err := n.Read(); err != nil {
			if err != io.EOF {
				return err
			} else {
				return nil
			}
		}
	}
}

func (n *NaimAV2) newSystemStatusResponse(data []byte) (SystemStatusResponse, error) {
	const dataLen = 4
	if len(data) != dataLen {
		return SystemStatusResponse{}, fmt.Errorf("expected length %v but received %v", dataLen, len(data))
	}
	arrData := (*[dataLen]byte)(data)
	return SystemStatusResponse{*arrData}, nil
}

func (ssr SystemStatusResponse) callHandlers(n *NaimAV2) error {
	return callHandlers(n.systemStatusCallbacks, ssr)
}

func (ssr SystemStatusResponse) Standby() bool {
	return ssr.status[0]&0x80 == 0
}

func (ssr SystemStatusResponse) Input() Input {
	input := Input(ssr.status[1] & 0x0f)
	switch input {
	case INPUT_VIP1:
		fallthrough
	case INPUT_VIP2:
		fallthrough
	case INPUT_OP1:
		fallthrough
	case INPUT_OP2:
		return input
	}
	return INPUT_UNKNOWN
}

func (ssr SystemStatusResponse) Muted() bool {
	return ssr.status[2]&0x80 > 0
}

func (ssr SystemStatusResponse) Volume() int {
	vol := ssr.status[2] & 0x7f
	if vol > 99 {
		vol = 99
	}
	return int(vol)
}

func (ur UnknownResponse) callHandlers(n *NaimAV2) error {
	return callHandlers(n.unknownResponseCallbacks, ur)
}

func callHandlers[T ~[]callback[S], S responseConstraint](cbs T, s S) error {
	for _, cb := range cbs {
		fmt.Printf("%v", cb)
		if err := cb(s); err != nil {
			return fmt.Errorf("error when calling SystemStatusResponse callback: %w", err)
		}
	}
	return nil
}
