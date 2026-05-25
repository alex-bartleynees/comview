package tui

// Run starts the comview TUI.
func Run(input string) error {
	rows, err := rowsForInput(input)
	if err != nil {
		return err
	}
	return runUIDiff(rows)
}
