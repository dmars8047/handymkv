package ui

import (
	"context"

	"github.com/rivo/tview"
)

type HandyPage int

const (
	Welcome HandyPage = iota
	Config
)

var pageNames = map[HandyPage]string{
	Welcome: "Welcome",
}

func (hp HandyPage) String() string {
	return pageNames[hp]
}

// PageNavigator is a page navigator
type PageNavigator struct {
	current    HandyPage
	Pages      *tview.Pages
	appContext context.Context
	openFuncs  map[HandyPage]func(interface{})
	closeFuncs map[HandyPage]func()
}

// NewNavigator creates a new page navigator
func NewNavigator(appContext context.Context) *PageNavigator {
	pages := tview.NewPages()

	return &PageNavigator{
		appContext: appContext,
		current:    Welcome,
		Pages:      pages,
		openFuncs:  make(map[HandyPage]func(interface{})),
		closeFuncs: make(map[HandyPage]func()),
	}
}

// Register registers a page with the page navigator
func (nav *PageNavigator) Register(page HandyPage,
	primitive tview.Primitive,
	resize, visible bool,
	openFunc func(interface{}),
	closeFunc func()) {

	nav.Pages.AddPage(page.String(), primitive, resize, visible)

	if openFunc != nil {
		nav.openFuncs[page] = openFunc
	}

	if closeFunc != nil {
		nav.closeFuncs[page] = closeFunc
	}
}

// NavigateTo navigates to a page
func (nav *PageNavigator) NavigateTo(page HandyPage, param interface{}) {
	close, ok := nav.closeFuncs[nav.current]

	if ok {
		close()
	}

	open, ok := nav.openFuncs[page]

	if ok {
		open(param)
	}

	nav.Pages.SwitchToPage(page.String())

	nav.current = page
}
