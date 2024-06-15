package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GridParam struct {
	row           int
	column        int
	rowSpan       int
	colSpan       int
	minGridHeight int
	minGridWidth  int
	focus         bool
}

type TUIPrimitive struct {
	tview.Primitive
	GridParameters GridParam
}

func MakeForm(
	dropdownCB func(string, int),
	searchCB, outputDirCB, depthCB func(string),
	limitCB func(bool),
	startCB, quitCB func(),
) *TUIPrimitive {
	form := tview.NewForm().
		SetFieldTextColor(tcell.ColorGhostWhite).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorOrangeRed).
		SetButtonTextColor(tcell.ColorOrangeRed).
		SetButtonBackgroundColor(tcell.ColorBlack).
		AddTextView("ArXiv Tree",
			"Welcome to ArXiv tree. Enter your search criteria below.\n"+
				"ArXiv may temporarily ban your IP if you send more than one\n"+
				"request every three seconds. Enable \"Avoid Rate Limit\" below\n"+
				"to circumvent this. You may also use a VPN for heavy loads.",
			0, 5, true, false).
		AddDropDown("Search by: ", []string{"ID", "Author", "Title"}, 2,
			dropdownCB,
		).
		AddTextArea("Search Query: ", "sample query", 0, 2, 0,
			searchCB,
		).
		AddTextArea("Output Dir: ", "arxiv-download-folder", 0, 1, 0,
			outputDirCB,
		).
		AddTextArea("Tree Depth: ", "1", 0, 1, 0,
			depthCB,
		).
		AddCheckbox("Avoid Rate Limit: ", false,
			limitCB,
		).
		AddButton("Start",
			startCB,
		).
		AddButton("Quit",
			quitCB,
		)
	form.SetBorder(true).
		SetTitle("Query").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(tcell.ColorOrangeRed).
		SetBorderColor(tcell.ColorGhostWhite)
	return &TUIPrimitive{
		Primitive: form,
		GridParameters: GridParam{
			row:           0,
			column:        0,
			rowSpan:       2,
			colSpan:       2,
			minGridHeight: 0,
			minGridWidth:  100,
			focus:         true,
		},
	}
}
