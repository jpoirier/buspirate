// Package buspirate interfaces with the binary mode of the BusPirate.
// http://dangerousprototypes.com/docs/Bus_Pirate
package buspirate

import (
	"fmt"
	"time"

	"github.com/jpoirier/serial"
)

// Open opens a connection to a BusPirate module and places it into binary mode.
func Open(dev string, readTimeout time.Duration) (*BusPirate, error) {
	t, err := serial.OpenPort(&serial.Config{Name: dev, Baud: 115200, ReadTimeout: readTimeout * time.Second})
	if err != nil {
		return nil, err
	}
	bp := BusPirate{t}
	return &bp, bp.enterBinaryMode()
}

// BusPirate represents a connection to a remote BusPirate device.
type BusPirate struct {
	*serial.Port
}

// enterBinaryMode resets the BusPirate and enters binary mode.
// http://dangerousprototypes.com/docs/Bitbang
func (bp *BusPirate) enterBinaryMode() error {
	bp.Flush()
	bp.Write([]byte{'\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n'})
	for i := 0; i < 30; i++ {
		// send binary reset
		if _, err := bp.Write([]byte{0x00}); err != nil {
			return fmt.Errorf("error, could not enter binary mode")
		}
		buf := make([]byte, 5)
		n, _ := bp.Read(buf)
		if n == 0 || string(buf) != "BBIO1" {
			return fmt.Errorf("error, could not enter binary mode")
		}
	}
	return nil
}

func (bp *BusPirate) leaveBinaryMode() error {
	if _, err := bp.Write([]byte{0x0F}); err != nil {
		return fmt.Errorf("error, could not leave binary mode")
	}
	time.Sleep(1000 * time.Millisecond)
	bp.Flush()
	return nil
}

// PowerOn turns on the 5v and 3v3 regulators.
func (bp *BusPirate) PowerOn() error {
	buf := []byte{0xC0}
	if _, err := bp.Write(buf); err != nil {
		return fmt.Errorf("error turning power on")
	}
	if n, _ := bp.Read(buf); n == 0 {
		return fmt.Errorf("error turning power on")
	}
	return nil
}

// PowerOff turns off the 5v and 3v3 regulators.
func (bp *BusPirate) PowerOff() error {
	buf := []byte{0x80}
	if _, err := bp.Write(buf); err != nil {
		return fmt.Errorf("error turning power off")
	}
	if n, _ := bp.Read(buf); n == 0 {
		return fmt.Errorf("error turning power off")
	}
	return nil
}

// SetPWM enables PWM output on the AUX pin with the specified duty cycle.
// duty is clamped between [0, 1].
func (bp *BusPirate) SetPWM(duty float64) error {
	clamp(&duty, 0.0, 1.0)
	PRy := uint16(0x3e7f)
	OCR := uint16(float64(PRy) * duty)
	buf := []byte{0x12, 0x00, uint8(OCR >> 8), uint8(OCR), uint8(PRy >> 8), uint8(PRy)}
	if _, err := bp.Write(buf); err != nil {
		return fmt.Errorf("error setting pwm")
	}
	if n, _ := bp.Read(buf[:1]); n == 0 {
		return fmt.Errorf("error setting pwm")
	}
	return nil
}

func clamp(v *float64, lower, upper float64) {
	if *v < lower {
		*v = lower
	}
	if *v > upper {
		*v = upper
	}
}

const (
	resetBitbangMode    = 0x00
	spiRawMode          = 0x01
	spiCSState          = 0x02
	spiBulkTransferMode = 0x10
	spiPeriphCfg        = 0x40
	spiSpeedCfg         = 0x60
	spiCfg              = 0x80
	spiWriteReadCmd     = 0x04
	spiWriteReadCmdNoCS = 0x05
	OneSecDelay         = 10000
)

// SpiEnter enters binary SPI mode.
func (bp *BusPirate) SpiEnter() error {
	if _, err := bp.Write([]byte{spiRawMode}); err != nil {
		return err
	}

	reply := make([]byte, 4)
	n, _ := bp.Read(reply)
	if n == 0 || string(reply) != "SPI1" {
		return fmt.Errorf("error entering SPI mode")
	}

	return nil
}

// SpiLeave exits SPI mode, returning to bitbang mode.
func (bp *BusPirate) SpiLeave() error {
	_, err := bp.Write([]byte{resetBitbangMode})
	return err
}

// SpiCS sets the chip select state.
// high = true, low = false
func (bp *BusPirate) SpiCS(high bool) error {
	// 00000010 – CS low (0)
	// 00000011 – CS high (1)
	buf := []byte{spiCSState} // default to disabled
	if high {
		buf[0] |= 0x01
	}
	if _, err := bp.Write(buf); err != nil {
		return err
	}

	n, _ := bp.Read(buf)
	if n == 0 || buf[0] != 0x01 {
		return fmt.Errorf("error setting chip select state")
	}

	return nil
}

// SpiCfgPeriph configures the spi peripherals.
// 0100wxyz – Configure peripherals, w=power, x=pullups, y=AUX, z=CS
func (bp *BusPirate) SpiCfgPeriph(power, pullups, aux, cs bool) error {
	buf := []byte{spiPeriphCfg}
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

	if _, err := bp.Write(buf); err != nil {
		return err
	}

	n, _ := bp.Read(buf)
	if n == 0 || buf[0] != 0x01 {
		return fmt.Errorf("error configuring spi peripherals")
	}

	return nil
}

// SpiSpeed is the SPI bus speed
type SpiSpeed uint8

// SpiSpeed is the SPI bus speed
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

// SpiSpeed sets SPI bus speed.
func (bp *BusPirate) SpiSpeed(speed SpiSpeed) error {
	buf := []byte{spiSpeedCfg}
	buf[0] |= byte(speed & 0x07)
	if _, err := bp.Write(buf); err != nil {
		return err
	}
	n, _ := bp.Read(buf)
	if n == 0 || buf[0] != 0x01 {
		return fmt.Errorf("error setting the SPI speed")
	}

	return nil
}

// SpiCfg configures the SPI bus.
// 1000wxyz – SPI config, w=output type, x=idle, y=clock edge, z=sample
// The CKP and CKE bits determine, on which edge of the clock, data transmission occurs.
// w= pin output HiZ(0)/3.3v(1)
// x=CKP clock idle phase (low=0)
// y=CKE clock edge (active to idle=1)
// z=SMP sample time (middle=0)
func (bp *BusPirate) SpiCfg(output33v, idle, edge, sample bool) error {
	buf := []byte{spiCfg}
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

	if _, err := bp.Write(buf); err != nil {
		return err
	}

	n, _ := bp.Read(buf)
	if n == 0 || buf[0] != 0x01 {
		return fmt.Errorf("error configuring the SPI settings")
	}

	return nil
}

// SpiSend sends from 1 to 16 bytes to the SPI device. Reads a byte
// for each byte sent.
func (bp *BusPirate) SpiSend(data []byte) ([]byte, error) {
	// send cmd and read reply
	// send 1 - 16 bytes reading a reply byte after each send
	l := len(data)
	if l < 1 || l > 16 {
		return nil, fmt.Errorf("error, length must be between 1 and 16 bytes")

	}

	buf := []byte{spiBulkTransferMode | byte(l-1)}
	if _, err := bp.Write(buf); err != nil {
		return nil, err
	}

	n, _ := bp.Read(buf)
	if n == 0 || buf[0] != 0x01 {
		return nil, fmt.Errorf("error setting send cmd mode")
	}

	out := make([]byte, l)
	for i := 0; i < l; i++ {
		if _, err := bp.Write(data[i : i+1]); err != nil {
			return nil, err
		}

		if n, _ := bp.Read(out[i : i+1]); n == 0 {
			return nil, fmt.Errorf("error reading byte send reply")
		}
	}

	return out, nil
}

// SpiWriteRead writes 0-4096 bytes and/or reads 0-4096 bytes.
func (bp *BusPirate) SpiWriteRead(outData, inData []byte) error {
	// write send count
	// write receive count
	//  if any, write out data
	// read status byte
	// if any, read in data
	outCnt := len(outData)
	if outCnt < 0 || outCnt > 4096 {
		return fmt.Errorf("error, invalid out data count (0-4096 bytes)")
	}

	inCnt := len(inData)
	if inCnt < 0 || inCnt > 4096 {
		return fmt.Errorf("error, invalid in data count (0-4096 bytes)")
	}

	// send the ReadWrite command
	buf := []byte{spiWriteReadCmd, 0}
	if _, err := bp.Write(buf[:1]); err != nil {
		return err
	}

	// out data count
	buf[1] = byte(outCnt)
	buf[0] = byte(outCnt >> 8)
	if _, err := bp.Write(buf); err != nil {
		return err
	}

	// in data count
	buf[1] = byte(inCnt)
	buf[0] = byte(inCnt >> 8)
	if _, err := bp.Write(buf); err != nil {
		return err
	}

	if _, err := bp.Write(outData); err != nil {
		return err
	}

	// check status
	n, _ := bp.Read(buf[:1])
	if n == 0 || buf[0] != 1 {
		return fmt.Errorf("error with write/read operation")
	}

	// in data
	if inCnt > 0 {
		n, _ := bp.Read(inData)
		if n == 0 {
			return fmt.Errorf("error with write/read operation")
		}
	}

	return nil
}
