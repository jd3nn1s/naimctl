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

	Volume  int
	Input   av2.Input
	Muted   bool
	Standby bool
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

func (n *Naim) handleSystemStatus(response av2.SystemStatusResponse) error {
	return nil
}

func (n *Naim) handleUnknownResponse(response av2.UnknownResponse) error {
	return nil
}

func (n *Naim) Start() {
	n.av2.AddSystemStatusHandler(n.handleSystemStatus)
	n.av2.AddUnknownResponseHandler(n.handleUnknownResponse)
	go n.av2.ReadAll()
}
