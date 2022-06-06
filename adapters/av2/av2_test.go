package av2

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// use standalone definition for tests to
const startMsg = "#AV2 "

func TestNaim_ReadSystemStatusResponseFailures(t *testing.T) {
	reader := &bytes.Buffer{}
	n := NewNaim(reader, nil)

	n.AddSystemStatusHandler(func(statusResponse SystemStatusResponse) error {
		assert.FailNow(t, "callback should not be called when there is an error parsing")
		return nil
	})

	reader.WriteString(startMsg)
	reader.Write([]byte{0xff})
	assert.ErrorContains(t, n.Read(), "message below minimum length", "truncated with no msg code should error")

	reader.WriteString("#HAH ")
	reader.Write([]byte{0x69, 0x81, 0x87, 0xb2, 0x02, 0xff})
	assert.ErrorContains(t, n.Read(), "unexpected message start", "wrong device id should error")

	reader.WriteString(startMsg)
	reader.Write([]byte{0x69, 0x81, 0x87, 0x02, 0xff})
	assert.ErrorContains(t, n.Read(), "expected length", "wrong data length for msg code should error")

	// different response code should not trigger system status callback
	reader.WriteString(startMsg)
	reader.Write([]byte{0x01, 0x81, 0x87, 0x02, 0xff})
}

func TestNaim_ReadSystemStatusResponse(t *testing.T) {
	reader := &bytes.Buffer{}
	n := NewNaim(reader, nil)

	var ssr *SystemStatusResponse
	n.AddSystemStatusHandler(func(statusResponse SystemStatusResponse) error {
		ssr = &statusResponse
		return nil
	})
	reader.WriteString(startMsg)
	reader.Write([]byte{0x69, 0x81, 0x87, 0xb2, 0x02, 0xff})
	assert.NoError(t, n.Read())

	assert.True(t, ssr.muted())
	assert.Equal(t, 50, ssr.volume())
	assert.False(t, ssr.standby())
	assert.Equal(t, ssr.input(), INPUT_OP1)

	reader.WriteString(startMsg)
	reader.Write([]byte{0x69, 0x01, 0x87, 0xb2, 0x02, 0xff})
	assert.NoError(t, n.Read())

	assert.True(t, ssr.standby())

	reader.WriteString(startMsg)
	reader.Write([]byte{0x69, 0x81, 0x08, 0xb2, 0x02, 0xff})
	assert.NoError(t, n.Read())
	assert.Equal(t, ssr.input(), INPUT_OP2)

	reader.WriteString(startMsg)
	reader.Write([]byte{0x69, 0x81, 0x08, 0x63, 0x02, 0xff})
	assert.NoError(t, n.Read())
	assert.False(t, ssr.muted())
	assert.Equal(t, ssr.volume(), 99)
}

func TestNaim_ReadUnknownResponse(t *testing.T) {
	reader := &bytes.Buffer{}
	n := NewNaim(reader, nil)

	var ur *UnknownResponse
	n.AddUnknownResponseHandler(func(unknownResponse UnknownResponse) error {
		ur = &unknownResponse
		return nil
	})

	const testCode byte = 0x02
	reader.WriteString(startMsg)
	reader.Write([]byte{testCode, 0xff})
	assert.NoError(t, n.Read())
	assert.Equal(t, Response(testCode), ur.MsgCode)

	reader.WriteString(startMsg)
	reader.Write([]byte{testCode, 0x12, 0x34, 0xff})
	assert.NoError(t, n.Read())
	assert.Equal(t, Response(testCode), ur.MsgCode)
	assert.Equal(t, []byte{0x12, 0x34}, ur.Data)
}
