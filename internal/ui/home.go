package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type HomePage struct {
}

func (*HomePage) Setup(app *tview.Application, nav *PageNavigator) {
	grid := tview.NewGrid()

	grid.SetRows(8, 8, 2, 8, 1, 1, 0).
		SetColumns(0, 78, 0)

	logo := tview.NewTextView()
	logo.SetTextAlign(tview.AlignCenter)

	logoText := "██╗  ██╗ █████╗ ███╗   ██╗██████╗ ██╗   ██╗\n"
	logoText += "██║  ██║██╔══██╗████╗  ██║██╔══██╗╚██╗ ██╔╝\n"
	logoText += "███████║███████║██╔██╗ ██║██║  ██║ ╚████╔╝ \n"
	logoText += "██╔══██║██╔══██║██║╚██╗██║██║  ██║  ╚██╔╝  \n"
	logoText += "██║  ██║██║  ██║██║ ╚████║██████╔╝   ██║   \n"
	logoText += "╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝    ╚═╝   \n\n"

	logo.SetText(logoText)

	startButton := tview.NewButton("Start").SetSelectedFunc(func() {
		// nav.NavigateTo(LOGIN_PAGE, nil)
	})

	configButton := tview.NewButton("Config").SetSelectedFunc(func() {
		// nav.NavigateTo(LOGIN_PAGE, nil)
	})

	exitButton := tview.NewButton("Exit").SetSelectedFunc(func() {
		app.Stop()
	})

	buttonGrid := tview.NewGrid()

	buttonGrid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()

		goRight := func() {
			if startButton.HasFocus() {
				app.SetFocus(configButton)
			} else if configButton.HasFocus() {
				app.SetFocus(exitButton)
			} else if exitButton.HasFocus() {
				app.SetFocus(startButton)
			}
		}

		goLeft := func() {
			if startButton.HasFocus() {
				app.SetFocus(exitButton)
			} else if configButton.HasFocus() {
				app.SetFocus(startButton)
			} else if exitButton.HasFocus() {
				app.SetFocus(configButton)
			}
		}

		// vim movement keys
		if key == tcell.KeyRune {
			switch event.Rune() {
			case 'l':
				goRight()
			case 'h':
				goLeft()
			}
		}

		if key == tcell.KeyTab || key == tcell.KeyRight {
			goRight()
		} else if key == tcell.KeyBacktab || key == tcell.KeyLeft {
			goLeft()
		} else if key == tcell.KeyEscape {
			app.Stop()
		}
		return event
	})

	tvInstructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	tvInstructions.SetText("Navigate with Tab and Shift+Tab")

	buttonGrid.SetRows(3, 1, 1).SetColumns(4, 0, 2, 0, 2, 0, 4)

	buttonGrid.AddItem(startButton, 0, 1, 1, 1, 0, 0, true).
		AddItem(configButton, 0, 3, 1, 1, 0, 0, false).
		AddItem(exitButton, 0, 5, 1, 1, 0, 0, false).
		AddItem(tvInstructions, 2, 1, 1, 5, 0, 0, false)

	grid.AddItem(logo, 1, 1, 1, 1, 0, 0, false).
		AddItem(buttonGrid, 3, 1, 1, 1, 0, 0, true)

	nav.Register(Welcome, grid, true, true, func(param interface{}) {}, nil)
}
