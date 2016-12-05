package buspirate

import (
	"fmt"

	"github.com/jpoirier/lsport"
)

// Open opens a connection to a BusPirate module and places it into binary mode.
func Open(dev string) (*BusPirate, error) {
	term, err := lsport.Open(dev)
	if err != nil {
		return nil, err
	}
	bp := BusPirate{term}
	return &bp, bp.enterBinaryMode()
}

// BusPirate represents a connection to a remote BusPirate device.
type BusPirate struct {
	*lsport.Term
}

// enterBinaryMode resets the BusPirate and enters binary mode.
// http://dangerousprototypes.com/docs/Bitbang
func (bp *BusPirate) enterBinaryMode() error {
	bp.Flush(lsport.BufBoth)
	for i := 0; i < 20; i++ {
		// send binary reset
		if n, err := bp.Write([]byte{0x00}); n == 0 || err != nil {
			return fmt.Errorf("error writing binary mode command")
		}
		if err := bp.Drain(); err != nil {
			return err
		}
		if n, err := bp.BlockingRead(buf, 10); n == 0 || err != nil {
			continue
		}
		buf := make([]byte, n)
		if string(buf) == "BBIO1" {
			return nil
		}
	}
	return fmt.Errorf("error, could not enter binary mode")
}

// LeaveBinaryMode exits binary mode.
func (bp *BusPirate) LeaveBinaryMode() error {
	if n, err := bp.BlockingWrite([]byte{0x0F}, 2000); n == 0 || err != nil {
		return fmt.Errorf("error, could not leave binary mode")
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if err := bp.Close(); err != nil {
		return err
	}
	return nil
}

// PowerOn turns on the 5v and 3v3 regulators.
func (bp *BusPirate) PowerOn() error {
	buf := []byte{0xC0}
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error turning power on")
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error turning power on")
	}
	return nil
}

// PowerOff turns off the 5v and 3v3 regulators.
func (bp *BusPirate) PowerOff() error {
	buf := []byte{0x80}
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error turning power off")
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil {
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
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error setting pwm")
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf[:1], 2000); n == 0 || err != nil {
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
)

// SpiEnter enters binary SPI mode.
func (bp *BusPirate) SpiEnter() error {
	if n, err := bp.BlockingWrite([]byte{spiRawMode}, 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	reply := make([]byte, 4)
	_, err := bp.BlockingRead(reply, 2000)
	if err != nil || string(reply) != "SPI1" {
		return fmt.Errorf("error entering SPI mode")
	}
	return nil
}

// SpiLeave exits SPI mode, returning to bitbang mode.
func (bp *BusPirate) SpiLeave() error {
	if n, err := bp.BlockingWrite([]byte{resetBitbangMode}, 2000); n == 0 || err != nil {
		return nil
	}
	bp.Drain()
	return nil
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
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
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

	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
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
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
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

	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
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
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return nil, err
	}
	if err := bp.Drain(); err != nil {
		return nil, err
	}

	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
		return nil, fmt.Errorf("error setting send cmd mode")
	}

	out := make([]byte, l)
	for i := 0; i < l; i++ {
		if n, err := bp.BlockingWrite(data[i:i+1], 2000); n == 0 || err != nil {
			return nil, err
		}
		if err := bp.Drain(); err != nil {
			return nil, err
		}

		if n, err := bp.BlockingRead(out[i:i+1], 2000); n == 0 || err != nil {
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
	if n, err := bp.BlockingWrite(buf[:1], 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	// out data count
	buf[1] = byte(outCnt)
	buf[0] = byte(outCnt >> 8)
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	// in data count
	buf[1] = byte(inCnt)
	buf[0] = byte(inCnt >> 8)
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	if n, err := bp.Write(outData); n == 0 || err != nil {
		return err
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	// check status
	if n, err := bp.BlockingRead(buf[:1], 2000); n == 0 || err != nil || buf[0] != 1 {
		return fmt.Errorf("error with write/read operation")
	}

	// in data
	if inCnt > 0 {
		// TODO: proper time for make 4096 bits
		if n, err := bp.BlockingRead(inData, 60*1000); n != inCnt || err != nil {
			return fmt.Errorf("error with write/read operation")
		}
	}

	return nil
}
