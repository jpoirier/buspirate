package buspirate

import (
	"fmt"
	"strings"
	"time"

	"github.com/jpoirier/lsport"
)

// BusPirate represents a connection to a Bus Pirate device.
type BusPirate struct {
	*lsport.Term
}

// V3
const (
	baudReply   = "Set serial port speed: (bps)\r\n 1. 300\r\n 2. 1200\r\n 3. 2400\r\n 4. 4800\r\n 5. 9600\r\n 6. 19200\r\n 7. 38400\r\n 8. 57600\r\n 9. 115200\r\n10. BRG raw value"
	expectBaudReply   = "10. BRG raw value"
	brgReply    = "Enter raw value for BRG"
	brgValReply = "Adjust your terminal\r\nSpace to continue"
	expectBrgValReply = "Space to continue"
)


// Open opens a connection to a Bus Pirate device and places it in binary mode.
// Supported baud rates in addition to the standard ones below 115200:
// 500000, 1000000, and non Windows 2000000
func Open(dev string, baudrate int) (*BusPirate, error) {
	// default baud rate is 115200 at boot-up
	term, err := lsport.Open(dev, 115200)
	if err != nil {
		return nil, err
	}
	if baudrate != 115200 {
		err = resetBPBaudrate(term, baudrate)
		if err != nil {
			return nil, err
		}
		err = term.SetBaudrate(baudrate)
		if err != nil {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)

		reply := make([]byte, 20)
		term.Write([]byte{0x20}) // space character to confirm the baud rate change
		term.BlockingRead(reply, 10)
	}
	bp := BusPirate{term}
	return &bp, bp.enterBinaryMode()
}

// resetBPBaudrate resets (non-volatile) the Bus Pirate's baud rate.
func resetBPBaudrate(term *lsport.Term, buadrate int) error {
	var brg string
	switch buadrate {
	case 500000:
		brg = "7\n"
	case 1000000:
		brg = "3\n"
	case 2000000:
		if runtime.GOOS == 'windows' {
			return fmt.Errorf("error, 2000000 baud rate not supported on Windows")
		}
		brg = "1\n"
	default:
		return fmt.Errorf("error, invalid reset baudrate: %d, must be 5000000|1000000|5000000", buadrate)
	}

	// baud rate mode
	if n, err := term.Write([]byte("b\n")); n == 0 || err != nil {
		return fmt.Errorf("error writing baudrate command, n: %d, %v", n, err)
	}
	if err := term.Drain(); err != nil {
		return err
	}
	reply := make([]byte, len(baudReply)+10)
	if n, err := term.BlockingRead(reply, 500); n == 0 || err != nil {
		return fmt.Errorf("error reading baudrate command reply, n: %d, %v", n, err)
	}
	if !strings.Contains(string(reply), expectBaudReply) {
		return fmt.Errorf("error, baudrate command reply is invalid")
	}

	// brg mode
	if n, err := term.Write([]byte("10\n")); n == 0 || err != nil {
		return fmt.Errorf("error writing brg command, n: %d, %v", n, err)
	}
	if err := term.Drain(); err != nil {
		return err
	}
	reply = make([]byte, len(brgReply)+10)
	if n, err := term.BlockingRead(reply, 500); n == 0 || err != nil {
		return fmt.Errorf("error reading brg command reply, n: %d, %v", n, err)
	}
	if !strings.Contains(string(reply), brgReply) {
		return fmt.Errorf("error, brg command reply is invalid")
	}

	// brg value
	if n, err := term.Write([]byte(brg)); n == 0 || err != nil {
		return fmt.Errorf("error writing brg value, n: %d, %v", n, err)
	}
	if err := term.Drain(); err != nil {
		return err
	}
	reply = make([]byte, len(brgValReply)+10)
	if n, err := term.BlockingRead(reply, 500); n == 0 || err != nil {
		return fmt.Errorf("error reading brg value reply, n: %d, %v", n, err)
	}
	if !strings.Contains(string(reply), expectBrgValReply) {
		return fmt.Errorf("error, brg value reply is invalid")
	}

	return nil
}

func (bp *BusPirate) enterBinaryMode() error {
	bp.Write([]byte{'\n', '\n', '\n'})
	bp.Flush(lsport.BufBoth)
	buf := make([]byte, 5)
	for i := 0; i < 30; i++ {
		// send binary reset
		if n, err := bp.Write([]byte{0x00}); n == 0 || err != nil {
			return fmt.Errorf("error writing binary mode command, n: %d, %v", n, err)
		}
		if err := bp.Drain(); err != nil {
			return err
		}
		if n, err := bp.BlockingRead(buf, 10); n == 0 || err != nil {
			continue
		}
		if string(buf) == "BBIO1" {
			return nil
		}
	}
	return fmt.Errorf("error, could not enter binary mode")
}

// CloseTerm closes the terminal connection to the Bus Pirate device.
func (bp *BusPirate) CloseTerm() error {
	return bp.Close()
}

// LeaveBinaryMode exits binary mode.
func (bp *BusPirate) LeaveBinaryMode() error {
	if n, err := bp.BlockingWrite([]byte{0x0F}, 2000); n == 0 || err != nil {
		return fmt.Errorf("error leaving binary mode, n: %d, %v", n, err)
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
		return fmt.Errorf("error turning power on, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error turning power on reply, n: %d, %v", n, err)
	}
	return nil
}

// PowerOff turns off the 5v and 3v3 regulators.
func (bp *BusPirate) PowerOff() error {
	buf := []byte{0x80}
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error turning power off, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error turning power off reply, n: %d, %v", n, err)
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
		return fmt.Errorf("error setting pwm, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf[:1], 2000); n == 0 || err != nil {
		return fmt.Errorf("error setting pwm reply, n: %d, %v", n, err)
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
		return fmt.Errorf("error writing enter spi mode, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	reply := make([]byte, 4)
	if n, err := bp.BlockingRead(reply, 2000); err != nil || string(reply) != "SPI1" {
		return fmt.Errorf("error reading enter spi mode, n: %d, %v", n, err)
	}
	return nil
}

// SpiLeave exits SPI mode, returning to bitbang mode.
func (bp *BusPirate) SpiLeave() error {
	if n, err := bp.BlockingWrite([]byte{resetBitbangMode}, 2000); n == 0 || err != nil {
		return fmt.Errorf("error writing leave spi mode, n: %d, %v", n, err)
	}
	bp.Drain()
	return nil
}

// SpiCS sets the chip select state.
// high = true, low = false
func (bp *BusPirate) SpiCS(high bool) error {
	// 00000010 – CS low (0)
	// 00000011 – CS high (1)
	buf := []byte{spiCSState}
	if high {
		buf[0] |= 0x01
	}
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error writing set spi cs, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
		return fmt.Errorf("error reading set spi cs reply, n: %d, %v", n, err)
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
		return fmt.Errorf("error writing spi periph cfg, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
		return fmt.Errorf("error reading spi periph cfg reply, n: %d, %v", n, err)
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
		return fmt.Errorf("error writing spi speed, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
		return fmt.Errorf("error reading spi speed reply, n: %d, %v", n, err)
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
		return fmt.Errorf("error writing spi cfg, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
		return fmt.Errorf("error reading spi cfg reply, n: %d, %v", n, err)
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
		return nil, fmt.Errorf("error, spi send length must be between 1 and 16 bytes")
	}

	buf := []byte{spiBulkTransferMode | byte(l-1)}
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return nil, fmt.Errorf("error writing bulk transfer mode, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return nil, err
	}
	if n, err := bp.BlockingRead(buf, 2000); n == 0 || err != nil || buf[0] != 0x01 {
		return nil, fmt.Errorf("error reading bulk transfer mode reply, n: %d, %v", n, err)
	}

	out := make([]byte, l)
	for i := 0; i < l; i++ {
		if n, err := bp.BlockingWrite(data[i:i+1], 2000); n == 0 || err != nil {
			return nil, fmt.Errorf("error writing bulk transfer data, n: %d, %v", n, err)
		}
		if err := bp.Drain(); err != nil {
			return nil, err
		}
		if n, err := bp.BlockingRead(out[i:i+1], 2000); n == 0 || err != nil {
			return nil, fmt.Errorf("error reading bulk transfer data reply, n: %d, %v", n, err)
		}
	}
	return out, nil
}

// SpiWriteRead writes 0-4096 bytes and/or reads 0-4096 bytes.
func (bp *BusPirate) SpiWriteRead(outData, inData []byte) error {
	// write send count
	// write receive count
	// write out-data if any
	// get write/read status
	// read in-data if any
	outCnt := len(outData)
	if outCnt < 0 || outCnt > 4096 {
		return fmt.Errorf("error, spi read/write out-data count (0-4096 bytes)")
	}

	inCnt := len(inData)
	if inCnt < 0 || inCnt > 4096 {
		return fmt.Errorf("error, spi read/write in-data count (0-4096 bytes)")
	}

	// send the ReadWrite command
	buf := []byte{spiWriteReadCmd, 0}
	if n, err := bp.BlockingWrite(buf[:1], 2000); n == 0 || err != nil {
		return fmt.Errorf("error, spi read/write in-data count (0-4096 bytes)")
	}
	if err := bp.Drain(); err != nil {
		return err
	}

	// out data count
	buf[1] = byte(outCnt)
	buf[0] = byte(outCnt >> 8)
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error writing out-data count, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	// in data count
	buf[1] = byte(inCnt)
	buf[0] = byte(inCnt >> 8)
	if n, err := bp.BlockingWrite(buf, 2000); n == 0 || err != nil {
		return fmt.Errorf("error writing in-data count, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	if n, err := bp.Write(outData); n == 0 || err != nil {
		return fmt.Errorf("error writing out-data, n: %d, %v", n, err)
	}
	if err := bp.Drain(); err != nil {
		return err
	}
	// check status
	if n, err := bp.BlockingRead(buf[:1], 2000); n == 0 || err != nil || buf[0] != 1 {
		return fmt.Errorf("error out/in data status, n: %d, %v", n, err)
	}
	// in data
	if inCnt > 0 {
		// TODO: proper time for make 4096 bits
		if n, err := bp.BlockingRead(inData, 60*1000); n < inCnt || err != nil {
			return fmt.Errorf("errorreading in-data, n: %d, %v", n, err)
		}
	}

	return nil
}
