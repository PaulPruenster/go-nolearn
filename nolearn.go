package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/eiannone/keyboard"
)

type Status string

const (
	Todo Status = "todo"
	Seen Status = "seen"
	Done Status = "done"
)

type Task struct {
	Text   string `json:"text"`
	Status Status `json:"status"`
}

type NoLearn struct {
	tasks    []Task
	cursor   int
	filename string
}

func newNoLearn(filename string) *NoLearn {
	return &NoLearn{
		tasks:    []Task{},
		cursor:   0,
		filename: filename,
	}
}

func (nl *NoLearn) load() error {
	file, err := os.Open(nl.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, start empty
		}
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&nl.tasks)
}

func (nl *NoLearn) save() error {
	file, err := os.Create(nl.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(nl.tasks)
}

func (nl *NoLearn) addTask(text string) {
	if text == "" {
		return
	}
	newTask := Task{Text: text, Status: Todo}
	nl.tasks = append(nl.tasks, newTask)
	nl.cursor = len(nl.tasks) - 1
}

func (nl *NoLearn) deleteCurrentTask() {
	if len(nl.tasks) == 0 || nl.cursor < 0 || nl.cursor >= len(nl.tasks) {
		return
	}

	// Remove the task at cursor position
	nl.tasks = append(nl.tasks[:nl.cursor], nl.tasks[nl.cursor+1:]...)

	// Adjust cursor position
	if len(nl.tasks) == 0 {
		nl.cursor = 0
	} else if nl.cursor >= len(nl.tasks) {
		nl.cursor = len(nl.tasks) - 1
	}
}

func (nl *NoLearn) moveCursorUp() {
	if nl.cursor > 0 {
		nl.cursor--
	}
}

func (nl *NoLearn) moveCursorDown() {
	if nl.cursor < len(nl.tasks)-1 {
		nl.cursor++
	}
}

func (nl *NoLearn) cycleStatusForward() {
	if len(nl.tasks) == 0 {
		return
	}
	currentTask := &nl.tasks[nl.cursor]
	switch currentTask.Status {
	case Todo:
		currentTask.Status = Seen
	case Seen:
		currentTask.Status = Done
		// case Done:
		// 	currentTask.Status = Todo
	}
}

func (nl *NoLearn) cycleStatusBackward() {
	if len(nl.tasks) == 0 {
		return
	}
	currentTask := &nl.tasks[nl.cursor]
	switch currentTask.Status {
	// case Todo:
	// 	currentTask.Status = Done
	case Seen:
		currentTask.Status = Todo
	case Done:
		currentTask.Status = Seen
	}
}

func (nl *NoLearn) display() {
	clearScreen()

	fmt.Println("Nolearn. Press 'q' or ESC to quit.")
	fmt.Println("Controls: e/↑=up, d/↓=down, f=cycle status forward, s=cycle status backward, n=new task, x=delete")

	if len(nl.tasks) == 0 {
		fmt.Println("No tasks. Press 'n' to add a new task.")
		return
	}

	fmt.Printf("Tasks (%d total):\n\n", len(nl.tasks))

	for i, task := range nl.tasks {
		prefix := "  "
		if i == nl.cursor {
			prefix = "▶ "
		}

		var statusMarker string
		switch task.Status {
		case Todo:
			statusMarker = "[ ]"
		case Seen:
			statusMarker = "[~]"
		case Done:
			statusMarker = "[✓]"
		}

		fmt.Printf("%s%s %s\n", prefix, statusMarker, task.Text)
	}
}

func (nl *NoLearn) handleInput(char rune, key keyboard.Key) (shouldQuit bool) {
	switch {
	case char == 'q' || key == keyboard.KeyEsc:
		return true

	case char == 'e' || key == keyboard.KeyArrowUp:
		nl.moveCursorUp()

	case char == 'd' || key == keyboard.KeyArrowDown:
		nl.moveCursorDown()

	case char == 'f':
		nl.cycleStatusForward()
		nl.save()

	case char == 's':
		nl.cycleStatusBackward()
		nl.save()

	case char == 'n':
		nl.promptForNewTask()
		nl.save()

	case char == 'x':
		nl.deleteCurrentTask()
		nl.save()
	}

	return false
}

func (nl *NoLearn) promptForNewTask() {
	showCursor()
	defer hideCursor()

	// Temporarily close keyboard for line input
	keyboard.Close()
	defer func() {
		// Reopen keyboard, ignore errors for cleanup
		keyboard.Open()
	}()

	fmt.Print("\r\nEnter new task: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)

	nl.addTask(text)
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// Hide cursor using ANSI escape code
func hideCursor() {
	fmt.Print("\033[?25l")
}

// Show cursor using ANSI escape code
func showCursor() {
	fmt.Print("\033[?25h")
}

func main() {
	filename := "tasks.json"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	nl := newNoLearn(filename)
	if err := nl.load(); err != nil {
		fmt.Printf("Error loading tasks: %v\n", err)
		return
	}

	if err := keyboard.Open(); err != nil {
		fmt.Printf("Error opening keyboard: %v\n", err)
		return
	}
	defer keyboard.Close()

	hideCursor()
	defer showCursor()

	for {
		nl.display()

		char, key, err := keyboard.GetKey()
		if err != nil {
			fmt.Printf("Error reading key: %v\n", err)
			break
		}

		if nl.handleInput(char, key) {
			break
		}
	}

	if err := nl.save(); err != nil {
		fmt.Printf("Error saving tasks: %v\n", err)
	} else {
		fmt.Println("Tasks saved successfully.")
	}
}
