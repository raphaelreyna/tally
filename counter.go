package main

import "os"
import "regexp"
import "unicode"
import "io"
import "fmt"
import "sort"
import "strconv"
import "encoding/json"
import "io/ioutil"
import "path/filepath"
import "text/tabwriter"

const (
	colorReset = "\033[0m"

	colorRed = "\033[31m"
	colorGreen = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan = "\033[36m"
	colorWhite = "\033[37m"
)

type record struct {
	Label string
	Rune rune
	Count uint64
}

type mode int

const (
	normal mode = iota
	label
	number
)

type counter struct {
	state map[rune]*record
	records []*record

	mode mode

	lastRune rune
	label string

	number string
	adding bool

	saveFile string
}

func newCounter() *counter {
	const regex string = "-([a-zA-Z])=([[:^space:]]+)"

	// Compile regex
	r := regexp.MustCompile(regex)

	c := &counter{
		state: map[rune]*record{},
		records: []*record{},
		adding: true,
	}
	// Parse flags: extract key:label mapping from flags / args
	for i, arg := range os.Args {
		if i == 0 { continue }
		parts := r.FindStringSubmatch(arg)
		if len(parts) < 2 {
			c.saveFile = arg
			continue
		}
		r := rune(parts[1][0])
		c.state[r] = &record{Label: parts[2], Rune: r}
		c.records = append(c.records, c.state[r])
	}


	return c
}

func (c *counter) inc(r rune, by uint64) {
	// Make sure the rune exists
	if _, exists := c.state[r]; !exists {
		c.state[r] = &record{Label: string(r), Rune: r}
		c.records = append(c.records, c.state[r])
	}

	c.state[r].Count += by
}

func (c *counter) dec(r rune, by uint64) {
	// Make sure the rune exists
	if _, exists := c.state[r]; !exists {
		c.state[r] = &record{Label: string(r), Rune: r}
		c.records = append(c.records, c.state[r])
	}

	if c.state[r].Count < by {
		c.state[r].Count = 0
		return
	}
	c.state[r].Count -= by
}

func (c *counter) relabel(r rune, new string) {
	// Make sure the rune exists
	if _, exists := c.state[r]; !exists {
		c.state[r] = &record{Rune: r}
		c.records = append(c.records, c.state[r])
	}

	c.state[r].Label = new
}

func (c *counter) handleRune(r rune) bool {
	switch {
	case r == 27: // esc key
		switch c.mode {
		case normal:
			return false
		case label:
			c.dec(c.lastRune, 1) // that last one didn't count

			c.mode = normal
			c.lastRune = 0
			c.label = ""
		case number:
			c.mode = normal
			c.number = ""
		}
		return true
	case r == 13: // enter/return key
		switch c.mode {
		case normal:
			return false
		case label:
			c.relabel(c.lastRune, c.label)
			c.dec(c.lastRune, 1) // that last one didn't count

			// Reset rune relabel state
			c.label = ""
			c.mode = normal
			c.lastRune = 0
		case number:
			x, err := strconv.ParseUint(c.number, 10, 64)
			if err == nil {
				c.dec(c.lastRune, 1) // that last one didn't count
				if c.adding {
					c.inc(c.lastRune, x)
				} else {
					c.dec(c.lastRune, x)
				}
			}
			c.mode = normal
			c.number = ""
		}
		return true
	case r == 61: // = key
		if c.mode == normal && c.lastRune != 0 {
			c.mode = label
			c.label = ""
			return true
		}
	case r == 43: // + key
		if c.mode != label && c.lastRune != 0 {
			c.mode = number
			c.number = ""
			c.adding = true
		}
		return true
	case r == 45: // - key
		if c.mode != label && c.lastRune != 0 {
			c.mode = number
			c.number = ""
			c.adding = false
		}
		return true
	case unicode.IsPrint(r):
		switch {
		case c.mode == normal:
			c.inc(r, 1)
			c.lastRune = r
		case c.mode == label:
			c.label += string(r)
		case c.mode == number && unicode.IsNumber(r):
			c.number += string(r)
		}
		return true
	}
	return false
}

func (c *counter) render(w io.Writer) {
	var display string

	switch c.mode {
	case normal:
	case label:
		rec := c.state[c.lastRune]
		display += fmt.Sprintf("relabel %s (%s) as: %s\r\n- - -\r\n",
			rec.Label, string(c.lastRune), c.label,
		)
	case number:
		rec := c.state[c.lastRune]
		if c.adding {
			display += fmt.Sprintf("%s (%s) = %d + %s\r\n- - -\r\n",
				rec.Label, string(c.lastRune), rec.Count, c.number,
			)
		} else {
			display += fmt.Sprintf("%s (%s) = %d - %s\r\n- - -\r\n",
				rec.Label, string(c.lastRune), rec.Count, c.number,
			)
		}
	}

	sort.Sort(c)

	for _, rec := range c.records {
		display += fmt.Sprintf("%s (%s):\t%d\r\n", rec.Label, string(rec.Rune), rec.Count)
	}

	writer := tabwriter.NewWriter(w, 20, 4, 0, []byte(" ")[0], tabwriter.TabIndent)

	display += "\nPress the '?' key for help.\r\n"

	writer.Write([]byte(display))

	writer.Flush()
}

func (c *counter) loadSave() error {
	if c.saveFile == "" { return nil }

	file, err := os.Open(c.saveFile)
	defer file.Close()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := json.NewDecoder(file).Decode(&c.state); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	c.records = []*record{}
	for _, v := range c.state {
		c.records = append(c.records, v)
	}

	return err
}

func (c *counter) writeSave() error {
	if c.saveFile == "" { return nil }

	file, err := ioutil.TempFile(filepath.Dir(c.saveFile), "")
	defer file.Close()
	if err != nil { return err }

	data := map[rune]record{}
	for k, v := range c.state {
		data[k] = *v
	}

	if err := json.NewEncoder(file).Encode(&data); err != nil {
		return err
	}
	file.Close()

	if err = os.Remove(c.saveFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.Rename(file.Name(), c.saveFile)
}

func (c *counter) Len() int {
	return len(c.records)
}

func (c *counter) Less(i, j int) bool {
	if c.records[i].Count == c.records[j].Count {
		return c.records[i].Label < c.records[j].Label
	}
	return c.records[i].Count > c.records[j].Count
}

func (c *counter) Swap(i, j int) {
	tmp := c.records[i]
	c.records[i] = c.records[j]
	c.records[j] = tmp
}
