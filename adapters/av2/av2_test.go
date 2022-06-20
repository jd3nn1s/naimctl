package av2

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests do not use constants from the implementation, but define the expected values again. The hardware does not
// change so there is no maintenance overhead and avoids the possibility of testing with an incorrect 'actual' value.
const responseStartMsg = "#AV2 "
const commandStartMsg = "**AV2 "

func TestNaimAV2_ReadSystemStatusResponseFailures(t *testing.T) {
	reader := &bytes.Buffer{}
	n := NewNaim(reader, nil)

	n.AddSystemStatusHandler(func(statusResponse SystemStatusResponse) error {
		assert.FailNow(t, "callback should not be called when there is an error parsing")
		return nil
	})

	reader.WriteString(responseStartMsg)
	reader.Write([]byte{0xff})
	assert.ErrorContains(t, n.Read(), "message below minimum length", "truncated with no msg code should error")

	reader.WriteString("#HAH ")
	reader.Write([]byte{0x69, 0x81, 0x87, 0xb2, 0x02, 0xff})
	assert.ErrorContains(t, n.Read(), "unexpected message start", "wrong device id should error")

	reader.WriteString(responseStartMsg)
	reader.Write([]byte{0x69, 0x81, 0x87, 0x02, 0xff})
	assert.ErrorContains(t, n.Read(), "expected length", "wrong data length for msg code should error")

	// different response code should not trigger system status callback
	reader.WriteString(responseStartMsg)
	reader.Write([]byte{0x01, 0x81, 0x87, 0x02, 0xff})
}

func TestNaimAV2_ReadSystemStatusResponse(t *testing.T) {
	reader := &bytes.Buffer{}
	n := NewNaim(reader, nil)

	var ssr *SystemStatusResponse
	n.AddSystemStatusHandler(func(statusResponse SystemStatusResponse) error {
		ssr = &statusResponse
		return nil
	})
	reader.WriteString(responseStartMsg)
	reader.Write([]byte{0x69, 0x81, 0x87, 0xb2, 0x02, 0xff})
	assert.NoError(t, n.Read())

	assert.True(t, ssr.Muted())
	assert.Equal(t, 50, ssr.Volume())
	assert.False(t, ssr.Standby())
	assert.Equal(t, ssr.Input(), INPUT_OP1)

	reader.WriteString(responseStartMsg)
	reader.Write([]byte{0x69, 0x01, 0x87, 0xb2, 0x02, 0xff})
	assert.NoError(t, n.Read())

	assert.True(t, ssr.Standby())

	reader.WriteString(responseStartMsg)
	reader.Write([]byte{0x69, 0x81, 0x08, 0xb2, 0x02, 0xff})
	assert.NoError(t, n.Read())
	assert.Equal(t, ssr.Input(), INPUT_OP2)

	reader.WriteString(responseStartMsg)
	reader.Write([]byte{0x69, 0x81, 0x08, 0x63, 0x02, 0xff})
	assert.NoError(t, n.Read())
	assert.False(t, ssr.Muted())
	assert.Equal(t, ssr.Volume(), 99)
}

func TestNaimAV2_ReadUnknownResponse(t *testing.T) {
	reader := &bytes.Buffer{}
	n := NewNaim(reader, nil)

	var ur *UnknownResponse
	n.AddUnknownResponseHandler(func(unknownResponse UnknownResponse) error {
		ur = &unknownResponse
		return nil
	})

	const testCode byte = 0x02
	reader.WriteString(responseStartMsg)
	reader.Write([]byte{testCode, 0xff})
	assert.NoError(t, n.Read())
	assert.Equal(t, Response(testCode), ur.MsgCode)

	reader.WriteString(responseStartMsg)
	reader.Write([]byte{testCode, 0x12, 0x34, 0xff})
	assert.NoError(t, n.Read())
	assert.Equal(t, Response(testCode), ur.MsgCode)
	assert.Equal(t, []byte{0x12, 0x34}, ur.Data)
}

func TestNaimAV2_SetInput(t *testing.T) {
	writer := &bytes.Buffer{}
	n := NewNaim(nil, writer)

	assert.NoError(t, n.SetInput(INPUT_OP1))
	assert.Equal(t, append([]byte(commandStartMsg), []byte{0x35, 0xff}...), writer.Bytes())
	writer.Reset()

	assert.NoError(t, n.SetInput(INPUT_VIP2))
	assert.Equal(t, append([]byte(commandStartMsg), []byte{0x30, 0xff}...), writer.Bytes())
}

func TestNaimAV2_SetVolume(t *testing.T) {
	writer := &bytes.Buffer{}
	n := NewNaim(nil, writer)

	volumeLevel := 50
	assert.NoError(t, n.SetVolume(50))
	assert.Equal(t, append([]byte(commandStartMsg), []byte{0x23, byte(volumeLevel), 0xff}...), writer.Bytes())
}

func TestNaimAV2_SetVolumeOutOfRange(t *testing.T) {
	writer := &bytes.Buffer{}
	n := NewNaim(nil, writer)

	assert.Error(t, n.SetVolume(100))
	assert.Error(t, n.SetVolume(-1))
}

func TestNaimAV2_SetMute(t *testing.T) {
	writer := &bytes.Buffer{}
	n := NewNaim(nil, writer)

	assert.NoError(t, n.SetMute(true))
	assert.Equal(t, append([]byte(commandStartMsg), []byte{0x24, 0xff}...), writer.Bytes())

	writer.Reset()
	assert.NoError(t, n.SetMute(false))
	assert.Equal(t, append([]byte(commandStartMsg), []byte{0x25, 0xff}...), writer.Bytes())
}

func TestNaimAV2_SetStandby(t *testing.T) {
	writer := &bytes.Buffer{}
	n := NewNaim(nil, writer)

	assert.NoError(t, n.SetStandby(true))
	assert.Equal(t, append([]byte(commandStartMsg), []byte{0x22, 0xff}...), writer.Bytes())

	writer.Reset()
	assert.NoError(t, n.SetStandby(false))
	assert.Equal(t, append([]byte(commandStartMsg), []byte{0x21, 0xff}...), writer.Bytes())
}

func TestNaimAV2_ReadAll(t *testing.T) {
	reader := &bytes.Buffer{}

	// two identical messages to be read
	reader.Write([]byte(responseStartMsg))
	reader.Write([]byte{0x69, 0x81, 0x08, 0xb2, 0x02, 0xff})
	reader.Write([]byte(responseStartMsg))
	reader.Write([]byte{0x69, 0x81, 0x08, 0xb2, 0x02, 0xff})

	n := NewNaim(reader, nil)

	handlerCalls := 0
	n.AddSystemStatusHandler(func(statusResponse SystemStatusResponse) error {
		handlerCalls++
		return nil
	})
	assert.NoError(t, n.ReadAll())
	assert.Equal(t, 2, handlerCalls)
}

func TestNaimAV2_TestDouble(t *testing.T) {
	td := testDouble{}
	_, err := td.Write([]byte(commandStartMsg))
	// too fast message after pre-amble should fail
	assert.ErrorContains(t, err, "preamble")

	td = testDouble{}
	_, err = td.Write([]byte(commandStartMsg)[0:1])
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 25)
	_, err = td.Write([]byte(commandStartMsg)[1:])
	assert.NoError(t, err)
	_, err = td.Write([]byte{0x22, 0xff})
	assert.NoError(t, err)
	_, err = td.Write([]byte(commandStartMsg))
	// command sent too soon after last command
	assert.ErrorContains(t, err, "message being received too soon")

	td = testDouble{}
	_, err = td.Write([]byte(commandStartMsg)[0:1])
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 25)
	_, err = td.Write([]byte(commandStartMsg)[1:])
	assert.NoError(t, err)
	_, err = td.Write([]byte{0x22, 0xff})
	assert.NoError(t, err)
	// minimum time between messages
	time.Sleep(time.Millisecond * 100)
	_, err = td.Write([]byte(commandStartMsg)[0:1])
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 25)
	_, err = td.Write([]byte(commandStartMsg)[1:])
	assert.ErrorContains(t, err, "preamble without delay")
}

func TestNaimAV2_SendTiming(t *testing.T) {
	n := NewNaim(nil, &testDouble{})
	assert.NoError(t, n.SetMute(true))
	assert.NoError(t, n.SetMute(true))
	assert.NoError(t, n.SetMute(true))
	// wait for all message timings to be past
	time.Sleep(time.Millisecond * 300)
	assert.NoError(t, n.SetMute(true))
	assert.NoError(t, n.SetMute(true))
	assert.NoError(t, n.SetMute(true))
}

const (
	MSG_AWAITING_PREAMBLE = iota
	MSG_HEADER
	MSG_STARTED
)

type testDouble struct {
	lastPreambleReceivedTime time.Time
	lastCommandReceivedTime  time.Time
	state                    int
}

func (t *testDouble) Reset() {
	t.state = MSG_AWAITING_PREAMBLE
	t.lastPreambleReceivedTime = time.Time{}
	t.lastCommandReceivedTime = time.Time{}
}

func (t *testDouble) Write(p []byte) (int, error) {
	now := time.Now()

	for _, b := range p {
		if t.state == MSG_AWAITING_PREAMBLE {
			if b != 0x2a {
				t.Reset()
				return 0, errors.New("expecting preamble")
			}
			if now.Before(t.lastCommandReceivedTime.Add(time.Millisecond * 100)) {
				t.Reset()
				return 0, errors.New("next message being received too soon")
			}
			t.lastPreambleReceivedTime = now
			t.state = MSG_HEADER
		} else if t.state == MSG_HEADER {
			if now.Before(t.lastPreambleReceivedTime.Add(time.Millisecond*25)) &&
				now.After(t.lastCommandReceivedTime.Add(time.Millisecond*200)) {
				t.Reset()
				return 0, errors.New("message received too soon after preamble")
			} else if now.Before(t.lastCommandReceivedTime.Add(time.Millisecond*200)) &&
				now.After(t.lastPreambleReceivedTime.Add(time.Millisecond*5)) {
				t.Reset()
				return 0, errors.New("message should follow preamble without delay, when received within 200ms")
			}
			t.lastCommandReceivedTime = now
			t.state = MSG_STARTED
		} else if t.state == MSG_STARTED {
			if b == 0xff {
				t.state = MSG_AWAITING_PREAMBLE
			}
		}
	}
	return len(p), nil
}

// confirm interface implemented
var _ io.Writer = &testDouble{}
