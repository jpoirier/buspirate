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
	"time"

	"github.com/jpoirier/buspirate"
)

func main() {
	fmt.Println("opening bp...")
	bp, err := buspirate.Open("/dev/ttyUSB0")
	// bp, err := buspirate.Open("/dev/ttyUSB0", 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}

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

	fmt.Println(r)
	time.Sleep(1000 * time.Millisecond)

	// ---
	//
	fmt.Println("sending block mode command...")
	err = bp.SpiCS(false)
	if err != nil {
		fmt.Println(err)
		return
	}

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

	fmt.Println(r)
	time.Sleep(1000 * time.Millisecond)

	// ---
	fmt.Println("sending sending/reading data block...")
	out2 := make([]byte, 100)
	for i := 0; i < 100; i++ {
		out2[i] = byte(i + 1)
	}
	in := make([]byte, 100)

	if err := bp.SpiWriteRead(out2, in); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("\nbye...")
}