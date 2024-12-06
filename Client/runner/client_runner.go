package clientrunner

import (
	"client/client"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/rivo/tview"
)

// ANSI color codes for styling
const (
	reset  = "\033[0m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	red    = "\033[31m"
)

var options = []string{
	"1. Send a chat request",
	"2. View chat requests",
}

type ClientRunner interface {
	Start()
}

type clientRunner struct {
	client *client.Client
}

func NewClientRunner() ClientRunner {
	return &clientRunner{client.GetClient()}
}

func (cr *clientRunner) Start() {
	cr.startup()
	cr.drawMenu()
}

func clearTerminal() {
	// Check the operating system
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func (cr *clientRunner) showLoadingBarWithInitialization(task string, initFunc func() <-chan error) error {
	// Call the initialization function and get the result channel
	resultChan := initFunc()

	fmt.Printf("%s", task)
	dots := ""

	for {
		select {
		case err, ok := <-resultChan:
			fmt.Printf("\r%s%s", task, "...")
			// Exit the loop if the channel is closed
			if !ok {
				fmt.Println(" done!")
				return nil
			}
			if err != nil {
				fmt.Println("Error!")
				return err
			}
			fmt.Println(" done!")
			return nil
		default:
			// Update the loading dots
			dots += "."
			if len(dots) > 3 {
				dots = "."
			}
			fmt.Printf("\r%s%s", task, dots)
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (cr *clientRunner) startup() {
	clearTerminal() // Clear terminal before showing the message
	fmt.Println(string(blue) + "Welcome to Low Latency Chat!" + reset)
	fmt.Println(string(green) + "Enter your username to begin:" + reset)
	fmt.Scanln(&cr.client.UserName)
	clearTerminal()
	fmt.Printf("Hello, %s! Let's get you setup...\n", cr.client.UserName)

	err := cr.showLoadingBarWithInitialization("Registering", cr.client.Register)
	if err != nil {
		fmt.Println(string(red) + "Failed to register client. Please try again later." + reset)
		return
	}
	err = cr.showLoadingBarWithInitialization("Fetching Servers", cr.client.Initialize)
	if err != nil {
		fmt.Println(string(red) + "Failed to fetch servers. Please try again later." + reset)
		return
	}
	time.Sleep(1 * time.Second)
	clearTerminal()
	fmt.Println(string(green) + "Setup complete! You're ready to chat." + reset)
	clearTerminal()
}

func (cr *clientRunner) drawMenu() {
	app := tview.NewApplication()
	list := tview.NewList().
		AddItem(options[0], "Begin a chat with another user!", 'a', nil).
		AddItem(options[1], "View your incoming message requests!", 'b', nil).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
			os.Exit(0)
		})
	if err := app.SetRoot(list, true).SetFocus(list).Run(); err != nil {
		panic(err)
	}
}

func (cr *clientRunner) showOptions() {
	cr.drawMenu()
}
