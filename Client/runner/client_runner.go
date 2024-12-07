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

var (
	ACK_CONN       = "ACK"
	MSG_REQ_SENT   = "REQ_SENT"
	AWAITING_REQ   = "AWAITING_REQ"
	USER_NOT_FOUND = "USER_NOT_FOUND"
	SERVER_ERROR   = "SERVER_ERROR"
	REQ_ACCEPTED   = "REQ_ACCEPTED"
)

type ClientRunner interface {
	Start()
}

type clientRunner struct {
	client *client.Client
	app    *tview.Application
	pages  *tview.Pages
}

func NewClientRunner() ClientRunner {
	return &clientRunner{client: client.GetClient()}
}

func (cr *clientRunner) Start() {
	cr.app = tview.NewApplication()
	cr.pages = tview.NewPages()
	cr.startup()
	cr.drawMenu()

	cr.app.SetRoot(cr.pages, true).Run()
}

// Deprecate after migrating everything to Tview
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

// Deprecate after migrating everything to Tview
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

// Deprecate after migrating everything to Tview
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
		os.Exit(1)
		return
	}
	err = cr.showLoadingBarWithInitialization("Fetching Servers", cr.client.Initialize)
	if err != nil {
		fmt.Println(string(red) + "Failed to fetch servers. Please try again later." + reset)
		os.Exit(1)
		return
	}
	time.Sleep(1 * time.Second)
	clearTerminal()
	fmt.Println(string(green) + "Setup complete! You're ready to chat." + reset)
	clearTerminal()
}

func (cr *clientRunner) drawMenu() {
	list := tview.NewList().
		AddItem(options[0], "Begin a chat with another user!", 'a', cr.beginChatPage).
		AddItem(options[1], "View your incoming message requests!", 'b', nil).
		AddItem("Quit", "Press to exit", 'q', func() {
			cr.app.Stop()
			os.Exit(0)
		})
	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetTitle("Main Menu").SetTitleAlign(tview.AlignCenter)
	frame.SetBorder(true)
	cr.pages.AddPage("menu", frame, true, true)
}

func (cr *clientRunner) beginChatPage() {
	usernameInput := tview.NewInputField().SetLabel("Enter username: ").SetFieldWidth(30).SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	frame := tview.NewFrame(tview.NewForm().
		AddFormItem(usernameInput).
		AddButton("Begin Chat", func() {
			// Begin chat logic
			cr.startMatchMaking(usernameInput.GetText())
		},
		).
		AddButton("Back", func() {
			cr.pages.SwitchToPage("menu")
		},
		))
	frame.SetTitle("Begin Chat").SetTitleAlign(tview.AlignCenter)
	frame.SetBorder(true)
	cr.pages.AddPage("beginChat", frame, true, true)
}

func waitForRequestAccepted(responseChannel chan string, accepted chan bool) {
	// TODO: Fix DRY
	for {
		select {
		case response := <-responseChannel:
			if response == REQ_ACCEPTED {
				accepted <- true
				return
			} else if response == AWAITING_REQ {
				accepted <- false
				time.Sleep(1 * time.Second)
			}
		default:
			accepted <- false
			time.Sleep(1 * time.Second)
		}
	}
}

func (cr *clientRunner) startMatchMaking(username string) {
	textView := tview.NewTextView().SetChangedFunc(func() { cr.app.Draw() })
	frame := tview.NewFrame(textView)
	frame.SetTitle("Matchmaking").SetTitleAlign(tview.AlignCenter)
	frame.SetBorder(true)
	cr.pages.AddAndSwitchToPage("matchmaking", frame, true)
	responseChannel := make(chan string)
	go cr.client.StartMatchmaking(username, responseChannel) // Make sure to run this in a goroutine
	go func() {
		text := ""
		text += "Waiting for server..."
		textView.SetRegions(true).SetText(text)
		response := <-responseChannel
		if response == ACK_CONN {
			text += " [green]Connected![white]\n"
			textView.SetText(text)
		} else {
			textView.Clear()
			textView.SetText("Failed to connect to server! [red]Please try again later")
			cr.pages.SwitchToPage("menu")
			// close the channel
			close(responseChannel)
			return
		}
		//time.Sleep(1 * time.Second)
		text += fmt.Sprintf("Sending chat request to %s...", username)
		textView.SetText(text)
		response = <-responseChannel
		if response == MSG_REQ_SENT {
			text += " [green]Sent![white]\n"
			textView.SetText(text)
		} else {
			textView.Clear()
			textView.SetText(fmt.Sprintf("[red] Failed to send chat request to %s! Please try again later", username))
			cr.pages.SwitchToPage("menu")
			// close the channel
			close(responseChannel)
			return
		}

		chatRequestAcceptedChannel := make(chan bool)
		go waitForRequestAccepted(responseChannel, chatRequestAcceptedChannel)
		// Show loading bar
		terminated := false
		dots := []string{".", "..", "...", "....", ".....", "......"}
		dotIdx := 0
		for !terminated {
			select {
			case chatAccepted := <-responseChannel:
				// Channel value read, exit the loop
				if chatAccepted == REQ_ACCEPTED {
					text += "Awaiting response... [green]Chat request accepted!"
					textView.SetText(text)
					terminated = true
					break
				} else if chatAccepted == SERVER_ERROR {
					text += "Awaiting response... [red]Chat request declined!"
					textView.SetText(text)
					terminated = true
					break
				} else if chatAccepted == AWAITING_REQ {
					dot := dots[dotIdx]
					loadingText := fmt.Sprintf("Awaiting response%s", dot)
					text += loadingText
					dotIdx++
					if dotIdx == len(dots) {
						dotIdx = 0
					}
					textView.SetText(text)
					/* Remove the added line */
					text = text[:len(text)-len(loadingText)]
				}
			default:
				// Do nothing
				continue
			}
			// debounce by 20 ms
			//time.Sleep(20 * time.Millisecond)
		}

		// close all channels
		close(responseChannel)
		close(chatRequestAcceptedChannel)
	}()
}

func (cr *clientRunner) showOptions() {
	cr.drawMenu()
	cr.beginChatPage()
}
