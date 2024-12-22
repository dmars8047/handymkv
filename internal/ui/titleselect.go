package ui

type TitleSelectPage struct {
	Id HandyPage
}

func NewTitleSelectPage() *TitleSelectPage {
	return &TitleSelectPage{
		Id: TitleSelect,
	}
}

// func (page *TitleSelectPage) Setup(app *tview.Application, nav *PageNavigator) {
// 	grid := tview.NewGrid()

// 	grid.SetRows(18, 6).SetColumns(2, 78, 2)

// 	go func() {
// 		titles := mkv.GetTitlesFromDisc()
// 	}

// }
