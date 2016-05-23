// Package buspirate interfaces with the binary mode of the BusPirate.
// http://dangerousprototypes.com/docs/Bus_Pirate
package buspirate

import (
	"fmt"
	"github.com/pkg/term"
	"time"
)

// Open opens a connection to a BusPirate module and places it into binary mode.
func Open(dev string) (*BusPirate, error) {
	t, err := term.Open(dev, term.Speed(115200), term.RawMode)
	if err != nil {
		return nil, err
	}
	bp := BusPirate{term: t}
	return &bp, bp.enterBinaryMode()
}

// BusPirate represents a connection to a remote BusPirate device.
type BusPirate struct {
	term *term.Term
}

// enterBinaryMode resets the BusPirate and enters binary mode.
// http://dangerousprototypes.com/docs/Bitbang
func (bp *BusPirate) enterBinaryMode() error {
	bp.term.Flush()
	bp.term.Write([]byte{'\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n'})
	const n = 30
	for i := 0; i < n; i++ {
		// send binary reset
		bp.term.Write([]byte{0x00})
		time.Sleep(10 * time.Millisecond)
		n, err := bp.term.Available()
		if err != nil {
			return err
		}
		buf := make([]byte, n)
		_, err = bp.term.Read(buf)
		if err != nil {
			return err
		}
		if string(buf) == "BBIO1" {
			return nil
		}
	}
	return fmt.Errorf("could not enter binary mode")
}

// PowerOn turns on the 5v and 3v3 regulators.
func (bp *BusPirate) PowerOn() {
	buf := []byte{0xc0}
	bp.term.Write(buf)
	bp.term.Read(buf)
}

// PowerOff turns off the 5v and 3v3 regulators.
func (bp *BusPirate) PowerOff() {
	buf := []byte{0x80}
	bp.term.Write(buf)
	bp.term.Read(buf)
}

// SetPWM enables PWM output on the AUX pin with the specified duty cycle.
// duty is clamped between [0, 1].
func (bp *BusPirate) SetPWM(duty float64) {
	clamp(&duty, 0.0, 1.0)
	PRy := uint16(0x3e7f)
	OCR := uint16(float64(PRy) * duty)
	buf := []byte{0x12, 0x00, uint8(OCR >> 8), uint8(OCR), uint8(PRy >> 8), uint8(PRy)}
	bp.term.Write(buf)
	bp.term.Read(buf[:1])
}

func (bp *BusPirate) SpiEnter() error {
	buf := []byte{0x01}
	reply := make([]byte, 4)
	bp.term.Write(buf)
	bp.term.Read(reply)
	if string(reply) == "SPI1" {
		return nil
	}
	return fmt.Errorf("Fail to enter SPI mode")
}

func (bp *BusPirate) SpiLeave() error {
	buf := []byte{0x00}
	reply := make([]byte, 5)
	bp.term.Write(buf)
	bp.term.Read(reply)
	if string(reply) == "BBIO1" {
		return nil
	}
	return fmt.Errorf("Fail to enter SPI mode")
}

/*
00001101 – Sniff all SPI traffic
00001110 – Sniff when CS low
00001111 – Sniff when CS high
*/

// 00000010 – CS low (0)
// 00000011 – CS high (1)
func (bp *BusPirate) SpiCs(high bool) error {
	buf := []byte{0x02}
	if high {
		buf[0] |= 0x01
	}
	bp.term.Write(buf)
	bp.term.Read(buf)
	if buf[0] == 0x01 {
		return nil
	}
	return fmt.Errorf("Set CS bad response")
}

// 0001xxxx – Bulk SPI transfer, send 1-16 bytes (0=1byte!)
func (bp *BusPirate) SpiTransfer(send []byte) (*[]byte, error) {
	l := len(send)
	if l < 1 || l > 16 {
		return nil, fmt.Errorf("Length must be beetween 1 and 16")
	}
	buf := []byte{0x10}
	buf[0] |= byte(l-1)
	bp.term.Write(buf)
	bp.term.Read(buf)
	if buf[0] != 0x01 {
		return nil, fmt.Errorf("Set CS bad response")
	}
	bp.term.Write(send)
	bp.term.Read(send)
	return &send, nil
}

// 0100wxyz – Configure peripherals, w=power, x=pullups, y=AUX, z=CS
func (bp *BusPirate) SpiConfigure(power, pullups, aux, cs bool) error {
	buf := []byte{0x40}
	if power {
		buf[0] |= 0x08
	}
	if pullups {
		buf[0] |= 0x04
	}
	if aux {
		buf[0] |= 0x02
	}
	if cs {
		buf[0] |= 0x01
	}
	bp.term.Write(buf)
	bp.term.Read(buf)
	if buf[0] == 0x01 {
		return nil
	}
	return fmt.Errorf("Set SPI config bad response")
}

type SpiSpeed uint8

const (
	SpiSpeed30khz SpiSpeed = iota
	SpiSpeed125khz
	SpiSpeed250khz
	SpiSpeed1mhz
	SpiSpeed2mhz
	SpiSpeed2600khz
	SpiSpeed4mhz
	SpiSpeed8mhz
)

// 01100xxx – Set SPI speed, 30, 125, 250khz; 1, 2, 2.6, 4, 8MHz
// 000=30kHz, 001=125kHz, 010=250kHz, 011=1MHz, 100=2MHz, 101=2.6MHz, 110=4MHz, 111=8MHz
func (bp *BusPirate) SpiSpeed(speed SpiSpeed) error {
	buf := []byte{0x60}
	buf[0] |= byte(speed & 0x07)
	bp.term.Write(buf)
	bp.term.Read(buf)
	if buf[0] == 0x01 {
		return nil
	}
	return fmt.Errorf("Set SPI speed bad response")
}

// 1000wxyz – SPI config, w=output type, x=idle, y=clock edge, z=sample
func (bp *BusPirate) SpiConfigure2(output33v, idle, edge, sample bool) error {
	buf := []byte{0x80}
	if output33v {
		buf[0] |= 0x08
	}
	if idle {
		buf[0] |= 0x04
	}
	if edge {
		buf[0] |= 0x02
	}
	if sample {
		buf[0] |= 0x01
	}
	bp.term.Write(buf)
	bp.term.Read(buf)
	if buf[0] == 0x01 {
		return nil
	}
	return fmt.Errorf("Set SPI config2 bad response")
}

func clamp(v *float64, lower, upper float64) {
	if *v < lower {
		*v = lower
	}
	if *v > upper {
		*v = upper
	}
}
