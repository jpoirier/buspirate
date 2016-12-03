package buspirate

import (
    import "time"

    import "github.com/jpoirier/buspirate"
)
// Pulse a LED connected to the AUX pin.
func ExampleBusPirate_SetPWM() {
	bp, err := buspirate.Open("/dev/ttyACM0", 5*time.Second)
	if err != nil {
		panic(err)
	}
	duty := 0.1
	delta := 0.1
	for {
		if err := bp.SetPWM(duty); err != nil {
			panic(err)
		}
		time.Sleep(50 * time.Millisecond)
		duty += delta
		if duty > 1.0 {
			duty = 1.0
			delta = -delta
		}
		if duty < 0 {
			duty = 0.0
			delta = -delta
		}
	}
}
