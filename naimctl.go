package naimctl

import (
	"fmt"
	"github.com/jd3nn1s/serial"
	"naimctl/adapters/av2"
	"time"
)

type Naim struct {
	av2  av2.NaimAV2
	port *serial.Port
}

func NewNaim(portDevice string) (*Naim, error) {
	port, err := serial.OpenPort(&serial.Config{
		Name:        portDevice,
		Baud:        9600,
		ReadTimeout: time.Second * 5,
		Size:        8,
		StopBits:    1,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to open RS485 port: %e", err)
	}

	return &Naim{
		av2:  av2.NewNaim(port, port),
		port: port,
	}, nil
}
