package av2

import (
	"errors"
	"fmt"
	"time"
)

func (n *NaimAV2) SetInput(input Input) error {
	var cmd Command
	switch input {
	case INPUT_OP1:
		cmd = COMMAND_INPUT_OP1
	case INPUT_OP2:
		cmd = COMMAND_INPUT_OP2
	case INPUT_VIP1:
		cmd = COMMAND_INPUT_VIP1
	case INPUT_VIP2:
		cmd = COMMAND_INPUT_VIP2
	default:
		return fmt.Errorf("unknown input %v", input)
	}
	return n.sendWithPreamble(cmd, nil)
}

func (n *NaimAV2) SetVolume(level int) error {
	if level < 0 || level > 99 {
		return errors.New("level must be between 0 and 99")
	}
	return n.sendWithPreamble(COMMAND_VOLUME, []byte{byte(level)})
}

func (n *NaimAV2) SetStandby(standby bool) error {
	if standby {
		return n.sendWithPreamble(COMMAND_STANDBY, nil)
	} else {
		return n.sendWithPreamble(COMMAND_ON, nil)
	}
}

func (n *NaimAV2) SetMute(mute bool) error {
	if mute {
		return n.sendWithPreamble(COMMAND_MUTE_ON, nil)
	} else {
		return n.sendWithPreamble(COMMAND_MUTE_OFF, nil)
	}
}

func (n *NaimAV2) sendWithPreamble(cmd Command, data []byte) error {
	durationSinceLastMessage := time.Since(n.lastSend)
	if durationSinceLastMessage < time.Millisecond*100 {
		// A delay of 100ms must be inserted between subsequent commands.
		time.Sleep(time.Millisecond*100 - durationSinceLastMessage)
	}

	if _, err := n.writer.Write([]byte("*")); err != nil {
		return err
	}

	// a small amount of time to send the command, as send has to complete before a deadline
	const durationToSendMessage = time.Millisecond * 5
	if durationSinceLastMessage > time.Millisecond*200-durationToSendMessage {
		// 25 ms sleep between preamble and message required
		time.Sleep(25 * time.Millisecond)
	} else if durationSinceLastMessage < time.Millisecond*100 {
		// A delay of 100ms must be inserted between subsequent commands.
		time.Sleep(time.Millisecond*100 - durationSinceLastMessage)
	} else {
		// If subsequent commands are sent within a 200ms window of each other than 25ms header delay is not required.
	}

	msg := []byte("*AV2 ")
	msg = append(msg, byte(cmd))
	if data != nil {
		msg = append(msg, data...)
	}
	msg = append(msg, 0xff)
	n.lastSend = time.Now()
	if _, err := n.writer.Write(msg); err != nil {
		return err
	}
	return nil
}
