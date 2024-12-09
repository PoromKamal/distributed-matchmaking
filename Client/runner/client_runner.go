package clientrunner

import (
	"client/client"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
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
	ACCEPT_REQ     = "ACCEPT_REQ"
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

func (cr *clientRunner) acceptChatRequest(username string) {
	textView := tview.NewTextView().SetChangedFunc(func() { cr.app.Draw() }).SetRegions(true)
	frame := tview.NewFrame(textView)
	frame.SetTitle(fmt.Sprintf("Accepting Chat Request with %s", username)).SetTitleAlign(tview.AlignCenter)
	frame.SetBorder(true)
	cr.pages.AddAndSwitchToPage("acceptingChatRequest", frame, true)
	go func() {
		statusChannel := make(chan string)
		defer close(statusChannel)

		go cr.client.AcceptMessageRequest(username, statusChannel)

		text := fmt.Sprintf("Accepting request from %s... ", username)
		textView.SetText(text)
		response := <-statusChannel
		if response == ACCEPT_REQ {
			text += "[green]Accepted![white]\n"
			textView.SetText(text)
		} else {
			textView.SetText("[red]Chat request declined! [red]You cannot chat with " + username + "[white]")
			time.Sleep(1 * time.Second)
			cr.pages.SwitchToPage("menu")
			return
		}
		text += "Awaiting for server matchmaking..."
		textView.SetText(text)
		response = <-statusChannel
		if response == SERVER_ERROR {
			textView.SetText("[red]Failed to connect to server! Please try again later.")
			time.Sleep(1 * time.Second)
			cr.pages.SwitchToPage("menu")
			return
		}
		roomId := <-statusChannel
		if roomId == SERVER_ERROR {
			textView.SetText("[red]Failed to connect to server! Please try again later.")
			time.Sleep(1 * time.Second)
			cr.pages.SwitchToPage("menu")
			return
		}

		text += "[green]Connected![white]\n"
		text += "Joining chat server on " + response
		textView.SetText(text)
		go cr.chatPage(response, roomId)
	}()
}

func (cr *clientRunner) drawMenu() {
	list := tview.NewList().
		AddItem(options[0], "Begin a chat with another user!", 'a', cr.beginChatPage).
		AddItem(options[1], "View your incoming message requests!", 'b', cr.beginChatRequestPage).
		AddItem("Quit", "Press to exit", 'q', func() {
			cr.app.Stop()
			os.Exit(0)
		})
	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetTitle("Main Menu").SetTitleAlign(tview.AlignCenter)
	frame.SetBorder(true)
	cr.pages.AddPage("menu", frame, true, true)
}

func (cr *clientRunner) beginChatRequestPage() {
	list := tview.NewList()
	list.AddItem("Back", "", 'q', func() {
		cr.pages.SwitchToPage("menu")
	})
	for username := range cr.client.ChatRequests {
		list.AddItem("Chat Request from: "+username, "", 0, func() { cr.acceptChatRequest(username) })
	}
	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetTitle("").SetTitleAlign(tview.AlignCenter)
	frame.SetBorder(true)
	cr.pages.AddAndSwitchToPage("chatRequests", frame, true)
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
	cr.pages.AddAndSwitchToPage("beginChat", frame, true)
}

func (cr *clientRunner) chatPage(serverAddr string, roomId string) {
	// Channel to receive chat messages
	messagesChannel := make(chan string)

	// Start the chat with the server
	go cr.client.StartChat(messagesChannel, serverAddr, roomId)

	// Create a text view to display the server name
	headerView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWrap(false).
		SetTextAlign(tview.AlignCenter).
		SetText("[cyan]Chatting on server: [white]" + cr.client.CurrentChatServer)

	// Create a text view to display chat messages
	chatView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWrap(true).
		SetChangedFunc(func() {
			cr.app.Draw()
		})

	// Create an input field for user input
	inputField := tview.NewInputField().
		SetLabel("Enter a message: ").
		SetFieldWidth(30)

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			// Get user input
			userMessage := inputField.GetText()
			cr.client.SendMessage(userMessage)
			inputField.SetText("")
		}
	})

	// Create a grid layout
	grid := tview.NewGrid().
		SetRows(1, 0, 3).                             // Header (fixed height), chat area (expandable), and input area (fixed height)
		SetColumns(0).                                // Full width
		AddItem(headerView, 0, 0, 1, 1, 0, 0, false). // Server name at the top
		AddItem(chatView, 1, 0, 1, 1, 0, 0, false).   // Chat messages in the middle
		AddItem(inputField, 2, 0, 1, 1, 0, 0, true)   // Input field at the bottom

	// Add the grid to pages and switch to it
	cr.pages.AddAndSwitchToPage("chat", grid, true)

	//Wait for the chat to start
	start := <-messagesChannel
	fmt.Println("Chat started with: ", start)
	if start != "START_CHAT" {
		cr.pages.SwitchToPage("menu")
		close(messagesChannel)
		return
	}

	// We know that cr.client.CurrentChatServer is hydrated for sure, so now we set it again
	headerView.SetText("[cyan]Chatting on server: [white]" + cr.client.CurrentChatServer)

	// Goroutine to listen to messages from the server
	go func() {
		text := ""
		for serverMessage := range messagesChannel {
			if strings.HasPrefix(serverMessage, cr.client.UserName) {
				text += "[yellow]" + serverMessage + "[white]\n"
			} else {
				text += "[green]" + serverMessage + "[white]\n"
			}
			chatView.SetText(text)
		}
	}()

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

		// Show loading bar
		terminated := false
		dots := []string{".", "..", "...", "....", ".....", "......"}
		dotIdx := 0
		for !terminated {
			select {
			case chatAccepted := <-responseChannel:
				// Channel value read, exit the loop
				if chatAccepted == REQ_ACCEPTED {
					text += "Awaiting response... [green]Chat request accepted![white]\n"
					textView.SetText(text)
					terminated = true
				} else if chatAccepted == SERVER_ERROR {
					text += "Awaiting response... [red]Chat request declined![white]\n"
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
				} else {
					textView.SetText("I HAVE NO IDEA WHAT JUST HAPPENED, here's what the server sent back: " + chatAccepted)
				}
			default:
				// Do nothing
				continue
			}
			// debounce by 20 ms
			//time.Sleep(20 * time.Millisecond)
		}

		server := <-responseChannel
		if server == SERVER_ERROR {
			textView.SetText("Failed to connect to server! [red]Please try again later[white]")
			cr.pages.SwitchToPage("menu")
			// close the channel
			close(responseChannel)
			return
		}

		if !strings.HasPrefix(server, "IP:") {
			textView.SetText("Failed to connect to server! [red]Please try again later[white]")
			cr.pages.SwitchToPage("menu")
			// close the channel
			close(responseChannel)
			return
		}

		serverAddress := strings.TrimPrefix(server, "IP:")

		roomId := <-responseChannel
		if roomId == SERVER_ERROR {
			textView.SetText("Failed to connect to server! [red]Please try again later[white]")
			cr.pages.SwitchToPage("menu")
			// close the channel
			close(responseChannel)
			return
		}
		if !strings.HasPrefix(roomId, "RoomID:") {
			textView.SetText("Failed to connect to server! [red]Please try again later[white]")
			cr.pages.SwitchToPage("menu")
			// close the channel
			close(responseChannel)
			return
		}
		roomId = strings.TrimPrefix(roomId, "RoomID:")
		roomId = strings.TrimSuffix(roomId, "\n")

		text += "Connecting to chat server on " + serverAddress
		textView.SetText(text)
		go cr.chatPage(serverAddress, roomId)
		close(responseChannel)
	}()
}
