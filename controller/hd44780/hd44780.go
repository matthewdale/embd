// Package hd44780 allows interfacing with a Hitachi HD44780-family
// LCD character display controller
//
// This library is based three HD44780 controller libraries
// Adafruit - https://github.com/adafruit/Adafruit-Raspberry-Pi-Python-Code/blob/master/Adafruit_CharLCD/Adafruit_CharLCD.py
// hwio - https://github.com/mrmorphic/hwio/blob/master/devices/hd44780/hd44780_i2c.go
// LiquidCrystal - https://github.com/arduino/Arduino/blob/master/libraries/LiquidCrystal/LiquidCrystal.cpp
//               - http://hmario.home.xs4all.nl/arduino/LiquidCrystal_I2C/
package hd44780

import (
	"time"

	"github.com/golang/glog"
	"github.com/kidoman/embd"
)

type entryMode byte
type displayMode byte
type functionMode byte

const (
	writeDelay = 37 * time.Microsecond
	pulseDelay = 1 * time.Microsecond
	clearDelay = 1520 * time.Microsecond

	// Initialize display
	lcdInit     byte = 0x33 // 00110011
	lcdInit4bit byte = 0x32 // 00110010

	// Commands
	lcdClearDisplay byte = 0x01 // 00000001
	lcdReturnHome   byte = 0x02 // 00000010
	lcdCursorShift  byte = 0x10 // 00010000
	lcdSetCGRamAddr byte = 0x40 // 01000000
	lcdSetDDRamAddr byte = 0x80 // 10000000

	// Cursor and display move flags
	lcdCursorMove  byte = 0x00 // 00000000
	lcdDisplayMove byte = 0x08 // 00001000
	lcdMoveLeft    byte = 0x00 // 00000000
	lcdMoveRight   byte = 0x04 // 00000100

	// Entry mode flags
	lcdSetEntryMode   entryMode = 0x04 // 00000100
	lcdEntryDecrement entryMode = 0x00 // 00000000
	lcdEntryIncrement entryMode = 0x02 // 00000010
	lcdEntryShiftOff  entryMode = 0x00 // 00000000
	lcdEntryShiftOn   entryMode = 0x01 // 00000001

	// Display mode flags
	lcdSetDisplayMode displayMode = 0x08 // 00001000
	lcdDisplayOff     displayMode = 0x00 // 00000000
	lcdDisplayOn      displayMode = 0x04 // 00000100
	lcdCursorOff      displayMode = 0x00 // 00000000
	lcdCursorOn       displayMode = 0x02 // 00000010
	lcdBlinkOff       displayMode = 0x00 // 00000000
	lcdBlinkOn        displayMode = 0x01 // 00000001

	// Function mode flags
	lcdSetFunctionMode functionMode = 0x20 // 00100000
	lcd4BitMode        functionMode = 0x00 // 00000000
	lcd8BitMode        functionMode = 0x10 // 00010000
	lcd1Line           functionMode = 0x00 // 00000000
	lcd2Line           functionMode = 0x08 // 00001000
	lcd5x8Dots         functionMode = 0x00 // 00000000
	lcd5x10Dots        functionMode = 0x04 // 00000100
)

// Represents a Hitachi HD44780-family character LCD controller
type HD44780 struct {
	HD44780Bus
	eMode entryMode
	dMode displayMode
	fMode functionMode
}

// NewGPIO4Bit creates a new HD44780 connected by a 4-bit GPIO bus
func NewGPIO4Bit(
	rs embd.DigitalPin,
	en embd.DigitalPin,
	d4 embd.DigitalPin,
	d5 embd.DigitalPin,
	d6 embd.DigitalPin,
	d7 embd.DigitalPin,
	backlight embd.DigitalPin,
	modes ...ModeSetter,
) (*HD44780, error) {
	pins := []embd.DigitalPin{rs, en, d4, d5, d6, d7, backlight}
	for _, pin := range pins {
		if pin == nil {
			continue
		}
		err := pin.SetDirection(embd.Out)
		if err != nil {
			glog.V(1).Infof("hd44780: error setting pin %+v to out direction: %s", pin, err)
			return nil, err
		}
	}
	return New(
		&GPIO4BitBus{
			RS:        rs,
			EN:        en,
			D4:        d4,
			D5:        d5,
			D6:        d6,
			D7:        d7,
			Backlight: backlight,
		},
		modes...,
	)
}

// NewI2C creates a new HD44780 connected by an I²C bus
func NewI2C(i2c embd.I2CBus, pinMap I2CPinMap, modes ...ModeSetter) (*HD44780, error) {
	return New(
		&I2CBus{
			I2C:       i2c,
			PinMap:    pinMap,
			Backlight: false,
		},
		modes...,
	)
}

func New(bus HD44780Bus, modes ...ModeSetter) (*HD44780, error) {
	controller := &HD44780{
		HD44780Bus: bus,
		eMode:      0x00,
		dMode:      0x00,
		fMode:      0x00,
	}
	err := controller.lcdInit()
	if err != nil {
		return nil, err
	}
	err = controller.SetMode(modes...)
	if err != nil {
		return nil, err
	}
	return controller, nil
}

func (controller *HD44780) lcdInit() error {
	glog.V(1).Info("hd44780: initializing display in 4-bit mode")
	err := controller.WriteInstruction(lcdInit)
	if err != nil {
		return err
	}
	return controller.WriteInstruction(lcdInit4bit)
}

type ModeSetter func(*HD44780)

// Entry mode modifiers
func EntryDecrement(x *HD44780) { x.eMode &= ^lcdEntryIncrement }
func EntryIncrement(x *HD44780) { x.eMode |= lcdEntryIncrement }
func EntryShiftOff(x *HD44780)  { x.eMode &= ^lcdEntryShiftOn }
func EntryShiftOn(x *HD44780)   { x.eMode |= lcdEntryShiftOn }

// Display mode modifiers
func DisplayOff(x *HD44780) { x.dMode &= ^lcdDisplayOn }
func DisplayOn(x *HD44780)  { x.dMode |= lcdDisplayOn }
func CursorOff(x *HD44780)  { x.dMode &= ^lcdCursorOn }
func CursorOn(x *HD44780)   { x.dMode |= lcdCursorOn }
func BlinkOff(x *HD44780)   { x.dMode &= ^lcdBlinkOn }
func BlinkOn(x *HD44780)    { x.dMode |= lcdBlinkOn }

// Function mode modifiers
func FourBitMode(x *HD44780)  { x.fMode &= ^lcd8BitMode }
func EightBitMode(x *HD44780) { x.fMode |= lcd8BitMode }
func OneLine(x *HD44780)      { x.fMode &= ^lcd2Line }
func TwoLine(x *HD44780)      { x.fMode |= lcd2Line }
func Dots5x8(x *HD44780)      { x.fMode &= ^lcd5x10Dots }
func Dots5x10(x *HD44780)     { x.fMode |= lcd5x10Dots }

// SetModes modifies the entry mode, display mode, and function modes with the given mode setter functions
func (controller *HD44780) SetMode(modes ...ModeSetter) error {
	for _, m := range modes {
		m(controller)
	}
	functions := []func() error{
		func() error { return controller.setEntryMode() },
		func() error { return controller.setDisplayMode() },
		func() error { return controller.setFunctionMode() },
	}
	for _, f := range functions {
		err := f()
		if err != nil {
			return err
		}
	}
	return nil
}

func (controller *HD44780) setEntryMode() error {
	return controller.WriteInstruction(byte(lcdSetEntryMode | controller.eMode))
}

func (controller *HD44780) setDisplayMode() error {
	return controller.WriteInstruction(byte(lcdSetDisplayMode | controller.dMode))
}

func (controller *HD44780) setFunctionMode() error {
	return controller.WriteInstruction(byte(lcdSetFunctionMode | controller.fMode))
}

// DisplayOff sets the display mode to off
func (controller *HD44780) DisplayOff() error {
	DisplayOff(controller)
	return controller.setDisplayMode()
}

// DisplayOn sets the display mode to on
func (controller *HD44780) DisplayOn() error {
	DisplayOn(controller)
	return controller.setDisplayMode()
}

// CursorOff sets the display mode to cursor off
func (controller *HD44780) CursorOff() error {
	CursorOff(controller)
	return controller.setDisplayMode()
}

// CursorOn sets the display mode to cursor on
func (controller *HD44780) CursorOn() error {
	CursorOn(controller)
	return controller.setDisplayMode()
}

// BlinkOff sets the display mode to cursor blink off
func (controller *HD44780) BlinkOff() error {
	BlinkOff(controller)
	return controller.setDisplayMode()
}

// BlinkOn sets the display mode to cursor blink on
func (controller *HD44780) BlinkOn() error {
	BlinkOn(controller)
	return controller.setDisplayMode()
}

// ShiftLeft shifts the cursor and all characters to the left
func (controller *HD44780) ShiftLeft() error {
	return controller.WriteInstruction(lcdCursorShift | lcdDisplayMove | lcdMoveLeft)
}

// ShiftRight shifts the cursor and all characters to the right
func (controller *HD44780) ShiftRight() error {
	return controller.WriteInstruction(lcdCursorShift | lcdDisplayMove | lcdMoveRight)
}

// Home moves the cursor and all characters to the home position
func (controller *HD44780) Home() error {
	err := controller.WriteInstruction(lcdReturnHome)
	time.Sleep(clearDelay)
	return err
}

// Clear clears the display and mode settings sets the cursor to the home position
func (controller *HD44780) Clear() error {
	err := controller.WriteInstruction(lcdClearDisplay)
	time.Sleep(clearDelay)
	return err
}

// SetCursor sets the input cursor to the given bye
func (controller *HD44780) SetCursor(value byte) error {
	return controller.WriteInstruction(lcdSetDDRamAddr | value)
}

// WriteInstruction writes a byte to the bus with register select in data mode
func (controller *HD44780) WriteChar(value byte) error {
	return controller.Write(true, value)
}

// WriteInstruction writes a byte to the bus with register select in command mode
func (controller *HD44780) WriteInstruction(value byte) error {
	return controller.Write(false, value)
}

func (controller *HD44780) Close() error {
	return controller.HD44780Bus.Close()
}

// ======== HD44780Bus Definitions ======== //
// ======================================== //

type HD44780Bus interface {
	// Write writes a byte to the HD44780 controller with the register select flag either on or off
	Write(rs bool, data byte) error

	// BacklightOff turns the optional backlight off
	BacklightOff() error

	// BacklightOn turns the optional backlight on
	BacklightOn() error

	// Close closes all open resources
	Close() error
}

// ============= GPIO4BitBus ============== //
// ======================================== //

type GPIO4BitBus struct {
	RS, EN         embd.DigitalPin
	D4, D5, D6, D7 embd.DigitalPin
	Backlight      embd.DigitalPin
}

func (bus *GPIO4BitBus) BacklightOff() error {
	if bus.Backlight != nil {
		return bus.Backlight.Write(embd.High)
	}
	return nil
}

func (bus *GPIO4BitBus) BacklightOn() error {
	if bus.Backlight != nil {
		return bus.Backlight.Write(embd.Low)
	}
	return nil
}

func (bus *GPIO4BitBus) Write(rs bool, data byte) error {
	rsInt := embd.Low
	if rs {
		rsInt = embd.High
	}
	functions := []func() error{
		func() error { return bus.RS.Write(rsInt) },
		func() error { return bus.D4.Write(int((data >> 4) & 0x01)) },
		func() error { return bus.D5.Write(int((data >> 5) & 0x01)) },
		func() error { return bus.D6.Write(int((data >> 6) & 0x01)) },
		func() error { return bus.D7.Write(int((data >> 7) & 0x01)) },
		func() error { return bus.pulseEnable() },
		func() error { return bus.D4.Write(int(data & 0x01)) },
		func() error { return bus.D5.Write(int((data >> 1) & 0x01)) },
		func() error { return bus.D6.Write(int((data >> 2) & 0x01)) },
		func() error { return bus.D7.Write(int((data >> 3) & 0x01)) },
		func() error { return bus.pulseEnable() },
	}
	for _, f := range functions {
		err := f()
		if err != nil {
			return err
		}
	}
	time.Sleep(writeDelay)
	return nil
}

func (bus *GPIO4BitBus) pulseEnable() error {
	values := []int{embd.Low, embd.High, embd.Low}
	for _, v := range values {
		time.Sleep(pulseDelay)
		err := bus.EN.Write(v)
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes all open pins
func (bus *GPIO4BitBus) Close() error {
	pins := []embd.DigitalPin{
		bus.RS,
		bus.EN,
		bus.D4,
		bus.D5,
		bus.D6,
		bus.D7,
		bus.Backlight,
	}

	for _, pin := range pins {
		err := pin.Close()
		if err != nil {
			glog.V(1).Infof("hd44780: error closing pin %+v: %s", pin, err)
			return err
		}
	}
	return nil
}

// ================ I2CBus ================ //
// ======================================== //

type I2CBus struct {
	I2C       embd.I2CBus
	PinMap    I2CPinMap
	Backlight bool
}

type I2CPinMap struct {
	RS, RW, EN     byte
	D4, D5, D6, D7 byte
	Backlight      byte
}

var (
	MJKDZPinMap I2CPinMap = I2CPinMap{
		RS: 6, RW: 5, EN: 4,
		D4: 0, D5: 1, D6: 2, D7: 3,
		Backlight: 7,
	}
	PCF8574PinMap I2CPinMap = I2CPinMap{
		RS: 0, RW: 1, EN: 2,
		D4: 4, D5: 5, D6: 6, D7: 7,
		Backlight: 3,
	}
)

const (
	i2cAddr byte = 0x00
)

func (bus *I2CBus) BacklightOff() error {
	bus.Backlight = false
	return bus.Write(false, 0x00)
}

func (bus *I2CBus) BacklightOn() error {
	bus.Backlight = true
	return bus.Write(false, 0x00)
}

func (bus *I2CBus) Write(rs bool, data byte) error {
	var instructionHigh byte = 0x00
	instructionHigh |= ((data >> 4) & 0x01) << bus.PinMap.D4
	instructionHigh |= ((data >> 5) & 0x01) << bus.PinMap.D5
	instructionHigh |= ((data >> 6) & 0x01) << bus.PinMap.D6
	instructionHigh |= ((data >> 7) & 0x01) << bus.PinMap.D7

	var instructionLow byte = 0x00
	instructionLow |= (data & 0x01) << bus.PinMap.D4
	instructionLow |= ((data >> 1) & 0x01) << bus.PinMap.D5
	instructionLow |= ((data >> 2) & 0x01) << bus.PinMap.D6
	instructionLow |= ((data >> 3) & 0x01) << bus.PinMap.D7

	instructions := []byte{instructionHigh, instructionLow}
	for _, ins := range instructions {
		if rs {
			ins |= 0x01 << bus.PinMap.RS
		}
		if bus.Backlight {
			ins |= 0x01 << bus.PinMap.Backlight
		}
		err := bus.pulseEnable(ins)
		if err != nil {
			return err
		}
	}
	time.Sleep(writeDelay)
	return nil
}

func (bus *I2CBus) pulseEnable(data byte) error {
	bytes := []byte{data, data | (0x01 << bus.PinMap.EN), data}
	for _, b := range bytes {
		time.Sleep(pulseDelay)
		err := bus.I2C.WriteByte(i2cAddr, b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bus *I2CBus) Close() error {
	return bus.I2C.Close()
}
