package hd44780

import (
	"testing"
	"time"

	"github.com/kidoman/embd"
)

const instructionTimeout = 1 * time.Second

type mockDigitalPin struct {
	direction embd.Direction
	values    chan int
	closed    bool
}

func newMockDigitalPin() *mockDigitalPin {
	return &mockDigitalPin{
		values: make(chan int, 256),
		closed: false,
	}
}

func (pin *mockDigitalPin) Write(val int) error {
	pin.values <- val
	return nil
}

func (pin *mockDigitalPin) SetDirection(dir embd.Direction) error {
	pin.direction = dir
	return nil
}

func (pin *mockDigitalPin) Close() error {
	pin.closed = true
	return nil
}

// unused
func (pin *mockDigitalPin) Watch(edge embd.Edge, handler func(embd.DigitalPin)) error { return nil }
func (pin *mockDigitalPin) StopWatching() error                                       { return nil }
func (pin *mockDigitalPin) N() int                                                    { return 0 }
func (pin *mockDigitalPin) Read() (int, error)                                        { return 0, nil }
func (pin *mockDigitalPin) TimePulse(state int) (time.Duration, error)                { return time.Duration(0), nil }
func (pin *mockDigitalPin) ActiveLow(b bool) error                                    { return nil }
func (pin *mockDigitalPin) PullUp() error                                             { return nil }
func (pin *mockDigitalPin) PullDown() error                                           { return nil }

type GPIO4BitBusEmulator struct {
	rs, en         *mockDigitalPin
	d4, d5, d6, d7 *mockDigitalPin
	backlight      *mockDigitalPin
	instructions   chan instruction
}

type instruction struct {
	rs   int
	data byte
}

func newGPIO4BitBusEmulator(t *testing.T) *GPIO4BitBusEmulator {
	be := &GPIO4BitBusEmulator{
		rs:           newMockDigitalPin(),
		en:           newMockDigitalPin(),
		d4:           newMockDigitalPin(),
		d5:           newMockDigitalPin(),
		d6:           newMockDigitalPin(),
		d7:           newMockDigitalPin(),
		backlight:    newMockDigitalPin(),
		instructions: make(chan instruction, 256),
	}
	go func() {
		for {
			var b byte = 0x00
			var rs int = 0
			if <-be.en.values == embd.Low &&
				<-be.en.values == embd.High &&
				<-be.en.values == embd.Low {
				rs = <-be.rs.values
				b |= byte(<-be.d4.values) << 4
				b |= byte(<-be.d5.values) << 5
				b |= byte(<-be.d6.values) << 6
				b |= byte(<-be.d7.values) << 7
			}
			if <-be.en.values == embd.Low &&
				<-be.en.values == embd.High &&
				<-be.en.values == embd.Low {
				b |= byte(<-be.d4.values)
				b |= byte(<-be.d5.values) << 1
				b |= byte(<-be.d6.values) << 2
				b |= byte(<-be.d7.values) << 3
				be.instructions <- instruction{rs, b}
			}
		}
	}()
	return be
}

func (be *GPIO4BitBusEmulator) pins() []*mockDigitalPin {
	return []*mockDigitalPin{be.rs, be.en, be.d4, be.d5, be.d6, be.d7, be.backlight}
}

func TestInitialize4Bit_directionOut(t *testing.T) {
	be := newGPIO4BitBusEmulator(t)
	NewGPIO4Bit(be.rs, be.en, be.d4, be.d5, be.d6, be.d7, be.backlight)
	for idx, pin := range be.pins() {
		if pin.direction != embd.Out {
			t.Errorf("Pin %d not set to direction Out", idx)
		}
	}
}

func TestInitialize4Bit_lcdInit(t *testing.T) {
	be := newGPIO4BitBusEmulator(t)
	NewGPIO4Bit(be.rs, be.en, be.d4, be.d5, be.d6, be.d7, be.backlight)
	instructions := []instruction{
		instruction{embd.Low, lcdInit},
		instruction{embd.Low, lcdInit4bit},
	}

	for idx, expected := range instructions {
		select {
		case ins := <-be.instructions:
			if ins.rs != expected.rs {
				t.Errorf("Instruction %d: Expected register select %d, got %d", idx, expected.rs, ins.rs)
			}
			if ins.data != expected.data {
				t.Errorf("Instruction %d: Expected byte %#x, got %#x", idx, expected.data, ins)
			}
		case <-time.After(instructionTimeout):
			t.Errorf("Instruction %d: Waited %s with no done signal", idx, instructionTimeout)
		}
	}
}

// func TestWrite(t *testing.T) {
// 	be := newGPIO4BitBusEmulator(t)
// 	expectedInstructions := make(chan instruction, 1)
// 	done := make(chan interface{}, 1)
// 	go func() {
// 		for {
// 			expected := <-expectedInstructions
// 			actual := <-be.instructions
// 			var bool found = false
// 			if actual.rs != expected.rs {
// 				t.Logf("Expected register select %d, got %d", expected.rs, actual.rs)
// 			} else {
// 				done <- nil
// 			}
// 			if actual.data != expected.data {
// 				t.Logf("Expected byte %#x, got %#x", expected.data, actual.data)
// 			} else {
// 				done <- nil
// 			}

// 		}
// 	}()
// 	controller, _ := NewGPIO4Bit(be.rs, be.en, be.d4, be.d5, be.d6, be.d7, be.backlight)
// 	cases := []map[string]interface{}{
// 		map[string]interface{}{
// 			"function": func() { controller.DisplayOn() },
// 			"expected": instruction{0x00, 0x00},
// 		},
// 	}
// 	for idx, c := range cases {
// 		function := c["function"].(func())
// 		expected := c["expected"].(instruction)
// 	}
// 	select {
// 	case <-done:
// 		continue
// 	case <-time.After(instructionTimeout):
// 		t.Errorf("Waited %s with no done signal", instructionTimeout)
// 	}
// }

func TestClose(t *testing.T) {
	be := newGPIO4BitBusEmulator(t)
	bus, _ := NewGPIO4Bit(be.rs, be.en, be.d4, be.d5, be.d6, be.d7, be.backlight)
	bus.Close()
	for idx, pin := range be.pins() {
		if !pin.closed {
			t.Errorf("Pin %d was not closed", idx)
		}
	}
}
