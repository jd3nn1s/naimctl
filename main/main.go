package main

import (
	"flag"
	"naimctl"
)

func main() {
	serialDeviceName := flag.String("serial", "/dev/ttyS0", "serial port device name")
	flag.Parse()

	n, err := naimctl.NewNaim(*serialDeviceName)
	if err != nil {
		panic(err)
	}
	_ = n
}
