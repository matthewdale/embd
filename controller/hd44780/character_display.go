package hd44780

import (
	"github.com/golang/glog"
	"github.com/kidoman/embd"
)

type CharacterDisplay struct {
	*HD44780
	Backlight BacklightBus
	Cols      int
	Rows      int
	p         *position
}

type position struct {
	col int
	row int
}

type BacklightBus interface {
	Off() error
	On() error
	Close() error
}

// NewCharacterDisplay initializes an easy-to-use character display based on the Hitachi HD44780
func NewCharacterDisplay(
	rs interface{},
	en interface{},
	d4 interface{},
	d5 interface{},
	d6 interface{},
	d7 interface{},
	backlight interface{},
	cols int,
	rows int,
	modes ...ModeSetter,
) (*CharacterDisplay, error) {
	pinKeys := []interface{}{rs, en, d4, d5, d6, d7, backlight}
	pins := [7]embd.DigitalPin{}
	for idx, key := range pinKeys {
		if key == nil {
			continue
		}
		var digitalPin embd.DigitalPin
		if pin, ok := key.(embd.DigitalPin); ok {
			digitalPin = pin
		} else {
			var err error
			digitalPin, err = embd.NewDigitalPin(key)
			if err != nil {
				glog.V(1).Infof("hd44780: error creating digital pin %+v: %s", key, err)
				return nil, err
			}
		}
		pins[idx] = digitalPin
	}
	defaultModes := []ModeSetter{
		FourBitMode,
		OneLine,
		Dots5x8,
		EntryIncrement,
		EntryShiftOff,
		DisplayOn,
		CursorOff,
		BlinkOff,
	}
	controller, err := NewGPIO4Bit(
		pins[0],
		pins[1],
		pins[2],
		pins[3],
		pins[4],
		pins[5],
		pins[6],
		append(defaultModes, modes...)...,
	)
	if err != nil {
		return nil, err
	}
	display := &CharacterDisplay{
		HD44780: controller,
		Cols:    cols,
		Rows:    rows,
		p:       &position{0, 0},
	}
	return display, display.BacklightOn()
}

// Home moves the cursor and all characters to the home position
func (disp *CharacterDisplay) Home() error {
	disp.currentPosition(0, 0)
	return disp.HD44780.Home()
}

// Clear clears the display, preserving the mode settings and setting the correct home
func (disp *CharacterDisplay) Clear() error {
	disp.currentPosition(0, 0)
	err := disp.HD44780.Clear()
	if err != nil {
		return err
	}
	err = disp.SetMode()
	if err != nil {
		return err
	}
	if !disp.isLeftToRight() {
		return disp.SetCursor(disp.Cols-1, 0)
	}
	return nil
}

// Message prints the given string on the display
func (disp *CharacterDisplay) Message(message string) error {
	bytes := []byte(message)
	for _, b := range bytes {
		if b == byte('\n') {
			err := disp.Newline()
			if err != nil {
				return err
			}
			continue
		}
		err := disp.WriteChar(b)
		if err != nil {
			return err
		}
		if disp.isLeftToRight() {
			disp.p.col++
		} else {
			disp.p.col--
		}
		if disp.p.col >= disp.Cols || disp.p.col < 0 {
			err := disp.Newline()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Newline moves the input cursor to the beginning of the next line
func (disp *CharacterDisplay) Newline() error {
	var col int
	if disp.isLeftToRight() {
		col = 0
	} else {
		col = disp.Cols - 1
	}
	return disp.SetCursor(col, disp.p.row+1)
}

func (disp *CharacterDisplay) isLeftToRight() bool {
	// same as (disp.HD44780.eMode&lcdEntryIncrement > 0) != (disp.HD44780.eMode&lcdEntryShiftOn > 0)
	return disp.HD44780.eMode>>1&0x01 != disp.HD44780.eMode&0x01
}

// SetCursor sets the input cursor to the given position
func (disp *CharacterDisplay) SetCursor(col, row int) error {
	if row >= disp.Rows {
		row = disp.Rows - 1
	}
	disp.currentPosition(col, row)
	return disp.HD44780.SetCursor(byte(col) + disp.lcdRowOffset(row))
}

func (disp *CharacterDisplay) lcdRowOffset(row int) byte {
	// Offset for up to 4 rows.
	if row > 3 {
		row = 3
	}
	switch disp.Cols {
	case 16:
		// 16-char line mappings
		return []byte{0x00, 0x40, 0x10, 0x50}[row]
	default:
		// default to the 20-char line mappings
		return []byte{0x00, 0x40, 0x14, 0x54}[row]
	}
}

func (disp *CharacterDisplay) currentPosition(col, row int) {
	disp.p.col = col
	disp.p.row = row
}

func (disp *CharacterDisplay) Close() error {
	err := disp.HD44780.Close()
	if err != nil {
		return err
	}
	return disp.Backlight.Close()
}
