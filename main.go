package main

import "os"
import "github.com/gosuri/uilive"
import "fmt"
import "bufio"
import "golang.org/x/crypto/ssh/terminal"

var Version string
var Date string

type flag uint

const (
	none flag = iota
	help
	version
)

func main() {
	retCode := 1
	defer func() {
		os.Exit(retCode)
	}()

	// Create counter, and get the flag that tells us if the user just wants the help text
	c, f := newCounter()
	switch {
	case f == version:
		fmt.Printf("version: %s\nbuild date: %s\n", Version, Date)
		retCode = 0
		return
	case f == help:
		fmt.Printf(`Usage: %s [LABEL]... [FILE]
Keep track of haw many times a key has been pressed.

Labels passed in as flags must have the form '-x=label' where x is a single character and label is the desired label.

Press ^C (Ctrl-C) to exit without saving; ^D to save (if using a save file) and exit.
To relabel a key, press the key to (re)label, press the = key, type in the new label and press enter/return again.
To subtract from or add more than 1 to a key/label, press the key you wish to add/subtract from, then press either the + or - key, the number you wish you add/subtract then press enter/return.
`, os.Args[0])
		retCode = 0
		return
	}

	// Needed for capturing single character user input
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
        	fmt.Printf("error while configuring terminal: %s\n", err)
		return
	}
	defer terminal.Restore(0, oldState)
	stdin := bufio.NewReader(os.Stdin)

	// Load from save file
	if err := c.loadSave(); err != nil {
		fmt.Printf("error while loading from %s: %s\n", c.saveFile, err)
		return
	}

	// start dynamically updating the users terminal
	writer := uilive.New()
	writer.Start()
	c.render(writer)

	var save bool
	// Loop over user key presses
mainLoop:
	for {
		char, _, err := stdin.ReadRune()
		if err != nil {
			fmt.Println(err)
			return
		}
		switch char {
		case 4: // ^D
			save = true
			fallthrough
		case 3: // ^C
			retCode = 0
			break mainLoop
		case 63: // ?
			fmt.Fprintf(writer, "(Help): Press ^C (Ctrl-C) to exit without saving; ^D to save (if using a save file) and exit.\r\nTo relabel a key, press the key to (re)label, press the = key, type in the new label and press enter/return again.\r\nTo subtract from or add more than 1 to a key/label, press the key you wish to add/subtract from, then press either the + or - key, the number you wish you add/subtract then press enter/return.\r\n")
			continue
		}

		c.handleRune(char)
		c.render(writer)
	}

	writer.Stop()

	if save {
		if err = c.writeSave(); err != nil {
			fmt.Printf("error while saving to %s: %s\n", c.saveFile, err)
			return
		}
	}

	retCode = 0
}
