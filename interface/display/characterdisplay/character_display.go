package characterdisplay

type Controller interface {
	WriteChar(byte) error
	SetCursor(col, row int) error
	IsLeftToRight() bool
	Cols() int
	Rows() int
	Home() error
	Clear() error
	Close() error
}

// CharacterDisplay represents an abstract character display and provides a
// convenience layer on top of the basic HD44780 library.
type CharacterDisplay struct {
	Controller
	p *position
}

type position struct {
	col int
	row int
}

// NewCharacterDisplay creates a new character display abstraction for an
// HD44780-compatible controller.
func New(controller Controller) *CharacterDisplay {
	return &CharacterDisplay{
		Controller: controller,
		p:          &position{0, 0},
	}
}

// Home moves the cursor and all characters to the home position.
func (disp *CharacterDisplay) Home() error {
	disp.setCurrentPosition(0, 0)
	return disp.Controller.Home()
}

// Clear clears the display, preserving the mode settings and setting the correct home.
func (disp *CharacterDisplay) Clear() error {
	disp.setCurrentPosition(0, 0)
	err := disp.Controller.Clear()
	if err != nil {
		return err
	}
	if !disp.IsLeftToRight() {
		return disp.SetCursor(disp.Cols()-1, 0)
	}
	return nil
}

// Message prints the given string on the display.
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
		if disp.IsLeftToRight() {
			disp.p.col++
		} else {
			disp.p.col--
		}
		if disp.p.col >= disp.Cols() || disp.p.col < 0 {
			err := disp.Newline()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Newline moves the input cursor to the beginning of the next line.
func (disp *CharacterDisplay) Newline() error {
	var col int
	if disp.IsLeftToRight() {
		col = 0
	} else {
		col = disp.Cols() - 1
	}
	return disp.SetCursor(col, disp.p.row+1)
}

// SetCursor sets the input cursor to the given position.
func (disp *CharacterDisplay) SetCursor(col, row int) error {
	if row >= disp.Rows() {
		row = disp.Rows() - 1
	}
	disp.setCurrentPosition(col, row)
	return disp.Controller.SetCursor(col, row)
}

func (disp *CharacterDisplay) setCurrentPosition(col, row int) {
	disp.p.col = col
	disp.p.row = row
}
