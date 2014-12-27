package hd44780

import (
	"testing"

	"github.com/kidoman/embd"
)

const (
	rows = 20
	cols = 4
)

func TestInitializeCharacterDisplay(t *testing.T) {
	var pins []*mockDigitalPin
	for i := 0; i < 7; i++ {
		pins = append(pins, newMockDigitalPin())
	}
	NewCharacterDisplay(
		pins[0],
		pins[1],
		pins[2],
		pins[3],
		pins[4],
		pins[5],
		pins[6],
		cols,
		rows,
	)
	for idx, pin := range pins {
		if pin.direction != embd.Out {
			t.Errorf("Pin %d not set to direction Out(%d), set to %d", idx, embd.Out, pin.direction)
		}
	}
}
