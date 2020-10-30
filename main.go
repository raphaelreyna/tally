package main

import "os"
import "github.com/gosuri/uilive"
import "fmt"
import "bufio"
import "golang.org/x/crypto/ssh/terminal"

func main() {
	retCode := 1
	defer func() {
		os.Exit(retCode)
	}()

	// start dynamically updating the users terminal
	writer := uilive.New()
	writer.Start()

	// Needed for capturing single character user input
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
        	fmt.Printf("error while configuring terminal: %s\n", err)
		return
	}
	defer terminal.Restore(0, oldState)
	stdin := bufio.NewReader(os.Stdin)

	// Create counter and show user the initial empty state
	c := newCounter()

	// Load from save file
	if err := c.loadSave(); err != nil {
		fmt.Printf("error while loading from %s: %s\n", c.saveFile, err)
		return
	}
	c.render(writer)

	// Loop over user key presses
	for {
		char, _, err := stdin.ReadRune()
		if err != nil {
			fmt.Println(err)
			return
		}
		if char == 3 { // ^C
			retCode = 0
			break
		}

		c.handleRune(char)
		c.render(writer)
	}

	writer.Stop()

	if err = c.writeSave(); err != nil {
		fmt.Printf("error while saving to %s: %s\n", c.saveFile, err)
		return
	}

	retCode = 0
}
