/*
Arduino Uno       Bus Pirate
-----------       ----------
MOSI pin 11       Gray
MISO pin 12       Black
SCK  pin 13       Purple
SS   pin 10       White
*/
package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/jpoirier/buspirate"
)

var baudrate int

func init() {
	if runtime.GOOS == "linux" {
		baudrate = 2000000
	} else if runtime.GOOS == "windows" {
		baudrate = 1000000
	} else {
		baudrate = 115200
	}
}

func main() {
	fmt.Println("opening bp...")
	bp, err := buspirate.Open("/dev/ttyUSB0", baudrate)
	// bp, err := buspirate.Open("/dev/ttyUSB0", 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer bp.CloseTerm()
	defer bp.LeaveBinaryMode()
	fmt.Println("serial port is open")

	if err := bp.SpiEnter(); err != nil {
		fmt.Println(err)
		return
	}

	defer bp.SpiLeave()
	fmt.Println("entered spi mode")

	if err := bp.SpiCS(true); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("CS high")

	if err := bp.SpiSpeed(buspirate.SpiSpeed1mhz); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("speed set")

	if err := bp.SpiCfg(true, false, false, false); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("spi cfg done")

	//---
	//
	fmt.Println("calling SpiSend...")
	out := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	err = bp.SpiCS(false)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1000 * time.Millisecond)

	r, err := bp.SpiSend(out)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = bp.SpiCS(true)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1000 * time.Millisecond)

	fmt.Println(r)

	// ---
	//
	fmt.Println("sending block mode command...")
	err = bp.SpiCS(false)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1000 * time.Millisecond)

	// spi slave cmd for block mode
	r, err = bp.SpiSend([]byte{0xff})
	if err != nil {
		fmt.Println(err)
		return
	}

	err = bp.SpiCS(true)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1000 * time.Millisecond)
	fmt.Println(r)

	// ---
	fmt.Println("sending sending/reading data block...")
	in := make([]byte, 100)
	out = make([]byte, 100)
	for i := 0; i < 100; i++ {
		out[i] = byte(i + 1)
	}

	if err := bp.SpiWriteRead(out, in); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(in)

	fmt.Println("\nbye...")
}
