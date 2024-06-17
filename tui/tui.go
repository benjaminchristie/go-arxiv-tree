package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	log "github.com/benjaminchristie/go-arxiv-tree/arxiv_logger"
	"github.com/benjaminchristie/go-arxiv-tree/comms"
	ratelimiter "github.com/benjaminchristie/go-arxiv-tree/rate_limiter"
	"github.com/benjaminchristie/go-arxiv-tree/tree"
	comps "github.com/benjaminchristie/go-arxiv-tree/tui/components"
	"github.com/rivo/tview"
)

const (
	FORM_IDX = 0
	LOG_IDX  = 1
	PDF_IDX  = 2
	LINE_IDX = 3
	NET_IDX  = 4
	TREE_IDX = 5

	PDF_ARR_IDX = 0
	LOG_ARR_IDX = 1
	NET_ARR_IDX = 2
)

type FormData struct {
	QueryType  string
	QueryValue string
	TreeDepth  int
	OutputDir  string
	SafeQuery  bool
}

type TUI struct {
	App            *tview.Application
	Components     []*comps.TUIPrimitive
	Comms          [][]comms.Comm
	Grid           *tview.Grid
	TreeHead       *tree.ArxivTree
	TreeUpdateChan chan bool
	UpdateChan     chan bool
	FormChan       chan FormData
}

func MakeTUI() *TUI {
	var t *TUI

	fData := FormData{
		QueryType:  "Title",
		QueryValue: "sample query",
		TreeDepth:  1,
		OutputDir:  "arxiv-download-folder",
		SafeQuery:  false,
	}
	onDropDown := func(s string, _ int) {
		fData.QueryType = s
	}
	onSearch := func(s string) {
		fData.QueryValue = s
	}
	onDir := func(s string) {
		fData.OutputDir = s
	}
	onDepth := func(s string) {
		var err error
		fData.TreeDepth, err = strconv.Atoi(s)
		if err != nil {
			fData.TreeDepth = 1
		}
	}
	onLimit := func(b bool) {
		fData.SafeQuery = b
	}
	onStart := func() {
		go func() {
			t.FormChan <- fData
		}()
	}
	onQuit := func() {
		t.App.Stop()
	}

	N_COMPONENTS := 6
	app := tview.NewApplication()
	grid := tview.NewGrid()

	treeUpdateChan := make(chan bool)
	updateChan := make(chan bool)
	formChan := make(chan FormData)

	components := make([]*comps.TUIPrimitive, N_COMPONENTS)

	tuiComms := make([][]comms.Comm, 3)
	tuiComms[PDF_ARR_IDX] = make([]comms.Comm, 2)
	tuiComms[NET_ARR_IDX] = make([]comms.Comm, 1)
	tuiComms[LOG_ARR_IDX] = make([]comms.Comm, 1)
	tuiComms[PDF_ARR_IDX][0] = *comms.MakeComm(0)
	tuiComms[PDF_ARR_IDX][1] = *comms.MakeComm(0, func(i interface{}) interface{} {
		return t.App.GetFocus() != components[TREE_IDX].Primitive
	},
	)
	tuiComms[NET_ARR_IDX][0] = *comms.MakeComm(0, func(i interface{}) interface{} {
		s, ok := i.(string)
		if !ok {
			log.Printf("Error casting to string in NET_ARR_IDX callback")
			return s
		}
		return api.NetData{
			Message: s[:min(len(s), 1024)],
			Size:    len(s),
		}
	})
	tuiComms[LOG_ARR_IDX][0] = *comms.MakeComm(0)

	components[FORM_IDX] = comps.MakeForm(onDropDown, onSearch, onDir, onDepth, onLimit, onStart, onQuit)
	components[LOG_IDX] = comps.MakeLogs(&tuiComms[LOG_ARR_IDX][0])
	components[PDF_IDX] = comps.MakePDFLogs(&tuiComms[PDF_ARR_IDX][0])
	components[LINE_IDX], components[NET_IDX] = comps.MakeNet(&tuiComms[NET_ARR_IDX][0])
	components[TREE_IDX] = comps.MakeTreeDisplayComponent(nil, &tuiComms[PDF_ARR_IDX][1])

	t = &TUI{
		App:            app,
		Components:     components,
		Grid:           grid,
		TreeUpdateChan: treeUpdateChan,
		UpdateChan:     updateChan,
		FormChan:       formChan,
		TreeHead:       nil,
		Comms:          tuiComms,
	}
	return t
}

func (t *TUI) Run() {
	log.Printf("Beginning Run")
	for _, c := range t.Components {
		log.Printf("Adding %v %p", c, c)
		comps.AddToGrid(t.Grid, c)
	}

	go func() {
		for {
			formData, ok := <-t.FormChan
			if !ok {
				log.Printf("Continuing")
				continue
			}
			t.formSubmit(formData)
		}
	}()

	log.Printf("Running")

	go func() {
		ticker := time.NewTicker(time.Second / 15) // run at 15Hz
		for {
			<-ticker.C
			t.App.Draw()
		}
	}()
	if err := t.App.SetRoot(t.Grid, true).EnableMouse(true).Run(); err != nil {
		t.App.Stop()
		panic(err)
	}
}

func (t *TUI) sendLogs(s string, v ...any) {
	log.Printf(s, v...)
	for _, c := range t.Comms[LOG_ARR_IDX] {
		c.Send(fmt.Sprintf(s, v...))
	}
}
func (t *TUI) sendPDFLogs(s string, v ...any) {
	log.Printf(s, v...)
	for _, c := range t.Comms[PDF_ARR_IDX] {
		c.Send(fmt.Sprintf(s, v...))
	}
}

func (t *TUI) formSubmit(f FormData) {
	var err error
	log.Printf("In form submit")
	go t.sendLogs("Parsing Query")

	if f.SafeQuery {
		ratelimiter.Enable()
	}
	defer func() {
		time.Sleep(1 * time.Second)
		t.sendLogs("Awaiting New Query")
	}()

	info := tree.ArxivTreeInfo{
		Title:      "",
		ID:         "",
		Author:     "",
		SourcePath: "",
		BibPath:    "",
	}

	query := api.QueryRequest{}

	switch f.QueryType {
	case "ID":
		query.IDList = f.QueryValue
	case "Author":
		query.Author = f.QueryValue
	case "Title":
		query.Title = f.QueryValue
	}

	log.Printf("Parsing query with parameters %s, depth: %d, output: %s", f.QueryValue, f.TreeDepth, f.OutputDir)
	err = tree.MakeInfoFromQuery(&info, query, true, t.Comms[NET_ARR_IDX]...)
	if err != nil {
		log.Print(err)
		t.sendLogs("Error: %s", err.Error())
		return
	}

	t.TreeHead = &tree.ArxivTree{
		Head:     nil,
		Children: nil,
		Value:    info,
	}
	// yuck
	t.Components[TREE_IDX].Primitive.(*comps.TreeDisplay).UpdateHead(t.TreeHead)

	err = os.MkdirAll(f.OutputDir, 0755)
	if err != nil {
		log.Print(err)
		t.sendLogs("Error: %s", err.Error())
		return
	}
	// callback to populateTree is goroutine
	tree.PopulateTree(t.TreeHead, f.TreeDepth,
		func(n *tree.ArxivTree) {
			go t.sendLogs("Populating Tree for %s", n.Value.(tree.ArxivTreeInfo).Title)
			t.downloadPDFhelper(n, f.OutputDir)
		},
		t.Comms[NET_ARR_IDX]...,
	)
}

func (t *TUI) downloadPDFhelper(n *tree.ArxivTree, outputDir string) {
	id := n.Value.(tree.ArxivTreeInfo).ID
	au := n.Value.(tree.ArxivTreeInfo).Author
	ti := n.Value.(tree.ArxivTreeInfo).Title
	if id != "" {
		formatted := fmt.Sprintf("%s/%s_%s.pdf", outputDir, strings.Replace(ti, "/", "", -1), id)
		err := api.DownloadPDF(id, formatted, t.Comms[NET_ARR_IDX]...)
		if err != nil {
			log.Print(err)
			t.sendLogs("Error: %s", err.Error())
			return
		}
		m := fmt.Sprintf("PDF: %.20s: %.60s", au, ti)
		t.sendPDFLogs(m)
	} else {
		m := fmt.Sprintf("Could not download PDF %.40s", ti)
		t.sendLogs(m)
	}
}
