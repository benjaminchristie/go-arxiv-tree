package tui

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"time"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	"github.com/benjaminchristie/go-arxiv-tree/tree"
	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
)

type FormData struct {
	QueryType  string
	QueryValue string
	TreeDepth  int
	OutputDir  string
	SafeQuery  bool
}

type TUI struct {
	App          *tview.Application
	Display      *TreeDisplay
	OnUpdate     chan bool
	OnFormSubmit chan FormData
	LogChan      chan string
	PdfChan      chan string
	NetChan      chan api.NetData
	Head         *tree.Node
}

var f *os.File

func init() {
	var err error
	f, err = os.OpenFile("arxiv-tree.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
}

func listen(t *TUI, onFormSubmitCB func(*TUI, FormData), onUpdateCB func(), onLogCB func(string), onPdfCB func(string), onNetCB func(api.NetData)) {
	for {
		select {
		case m0 := <-t.OnFormSubmit:
			go onFormSubmitCB(t, m0)
		case m1 := <-t.LogChan:
			go onLogCB(m1)
		case m2 := <-t.PdfChan:
			go onPdfCB(m2)
		case <-t.OnUpdate:
			go onUpdateCB()
		case m3 := <-t.NetChan:
			go onNetCB(m3)
		}
	}
}

func formSubmit(t *TUI, f FormData) {
	var err error
	t.LogChan <- "Initializing Search Tree"

	defer func() {
		time.Sleep(1 * time.Second)
		t.LogChan <- "Awaiting new Query"
	}()

	switch f.QueryType {
	case "ID":
		if f.SafeQuery {
			t.Head, err = tree.SafeTuiMakeNodeFromID(f.QueryValue, t.NetChan)
		} else {
			t.Head, err = tree.TuiMakeNodeFromID(f.QueryValue, t.NetChan)
		}
		if err != nil {
			log.Print(err)
			t.LogChan <- fmt.Sprintf("Error: %s", err.Error())
			return
		}
		break
	case "Author":
		if f.SafeQuery {
			t.Head, err = tree.SafeTuiMakeNodeFromAuthor(f.QueryValue, t.NetChan)
		} else {
			t.Head, err = tree.TuiMakeNodeFromAuthor(f.QueryValue, t.NetChan)
		}
		if err != nil {
			log.Print(err)
			t.LogChan <- fmt.Sprintf("Error: %s", err.Error())
			return
		}
		break
	case "Title":
		if f.SafeQuery {
			t.Head, err = tree.SafeTuiMakeNodeFromTitle(f.QueryValue, t.NetChan)
		} else {
			t.Head, err = tree.TuiMakeNodeFromTitle(f.QueryValue, t.NetChan)
		}
		if err != nil {
			log.Print(err)
			t.LogChan <- fmt.Sprintf("Error: %s", err.Error())
			return
			// t.App.Stop()
		}
		break
	default:
		log.Print("hit default")
		t.App.Stop()
	}

	t.Display = updateHead(t.Display, t.Head)

	err = os.MkdirAll(f.OutputDir, 0755)
	if err != nil {
		t.LogChan <- "Could not create directory " + f.OutputDir
		return
	}
	downloadpdf := func(n *tree.Node) {
		var localErr error
		if n.Info.ID != "" {
			formatted := fmt.Sprintf("%s/%s_%s.pdf", f.OutputDir, strings.Replace(n.Info.Title, "/", "", -1), n.Info.ID)
			if f.SafeQuery {
				localErr = api.SafeTuiDownloadPDF(n.Info.ID, formatted, t.NetChan)
			} else {
				localErr = api.TuiDownloadPDF(n.Info.ID, formatted, t.NetChan)
			}
			if localErr != nil {
				t.PdfChan <- fmt.Sprintf("Error downloading PDF %.40s", n.Info.Title)
				return
			}
			m := fmt.Sprintf("PDF: %.20s: %.60s", n.Info.Author, n.Info.Title)
			t.PdfChan <- m
		} else {
			t.PdfChan <- fmt.Sprintf("Could not download PDF %.40s", n.Info.Title)
		}
	}
	if f.SafeQuery {
		tree.SafeAsyncLoggingPopulateTree(t.Head, f.TreeDepth, t.LogChan, t.NetChan, downloadpdf)
	} else {
		tree.AsyncLoggingPopulateTree(t.Head, f.TreeDepth, t.LogChan, t.NetChan, downloadpdf)
	}
	// tree.Traverse(t.Head, downloadpdf)
	// t.LogChan <- "Outputing graph view to %s. Run `dot -Tsvg output.gv -o <file>` to view."
	// tree.Visualize(t.Head, "output.gv")
}

func Run() {
	defer f.Close()
	queryType, searchQuery, saveDir := "Title", "sample query", "arxiv-download-folder"
	treeDepth := 1
	safeQuery := false
	t := &TUI{
		App:          tview.NewApplication(),
		OnUpdate:     make(chan bool, 100),
		OnFormSubmit: make(chan FormData, 100),
		LogChan:      make(chan string, 100),
		PdfChan:      make(chan string, 100),
		NetChan:      make(chan api.NetData, 100),
		Head:         nil,
		Display:      makeTreeDisplay(nil),
	}
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
			func(option string, _ int) {
				queryType = option
			},
		).
		AddTextArea("Search Query: ", "sample query", 0, 2, 0,
			func(text string) {
				searchQuery = text
			},
		).
		AddTextArea("Output Dir: ", "arxiv-download-folder", 0, 1, 0,
			func(text string) {
				saveDir = text
			},
		).
		AddTextArea("Tree Depth: ", "1", 0, 1, 0,
			func(text string) {
				var err error
				treeDepth, err = strconv.Atoi(text)
				if err != nil {
					treeDepth = 1
				}
			},
		).
		AddCheckbox("Avoid Rate Limit: ", false,
			func(checked bool) {
				safeQuery = checked
			},
		).
		AddButton("Start", func() {
			go func() {
				f := FormData{
					QueryType:  queryType,
					QueryValue: searchQuery,
					OutputDir:  saveDir,
					TreeDepth:  treeDepth,
					SafeQuery:  safeQuery,
				}
				t.OnFormSubmit <- f
			}()
		}).
		AddButton("Quit", func() {
			t.App.Stop()
			log.Fatalf("Exited program")
		})
	form.SetBorder(true).
		SetTitle("Query").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(tcell.ColorOrangeRed).
		SetBorderColor(tcell.ColorGhostWhite)

	grid := tview.NewGrid()

	logs := tview.NewTable()
	logs.SetBorder(true).
		SetTitle("Logs").
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitleColor(tcell.ColorGhostWhite).
		SetTitleAlign(tview.AlignCenter)
	pdfs := tview.NewTable()
	pdfs.SetBorder(true).
		SetTitle("PDFs").
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitleColor(tcell.ColorGhostWhite).
		SetTitleAlign(tview.AlignCenter)

	netPage := tview.NewTextArea()
	netPage.SetBorder(true).
		SetTitle("Network").
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitleColor(tcell.ColorGhostWhite).
		SetTitleAlign(tview.AlignCenter)

		// 	sparkLineIO := tvxwidgets.NewSparkline()
		// 	sparkLineIO.SetBorder(true).
		// 		SetTitle("Disk IO")

	sparkLineNet := tvxwidgets.NewSparkline()
	sparkLineNet.SetBorder(true).
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitle("Network IO")

	_, _, sparklineNetWidth, _ := sparkLineNet.GetInnerRect()
	sparklineNetWidth *= 8

	t.Display.TviewTree.SetBorder(true).
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitle("Current Tree")

	grid.AddItem(logs, 2, 2, 1, 2, 0, 100, false).
		AddItem(pdfs, 3, 2, 1, 2, 0, 100, false).
		AddItem(form, 0, 0, 2, 2, 0, 100, true).
		AddItem(netPage, 2, 0, 1, 2, 0, 100, false).
		AddItem(sparkLineNet, 3, 0, 1, 2, 0, 100, false).
		AddItem(t.Display.TviewTree, 0, 2, 2, 2, 0, 100, false)
		// AddItem(sparkLineIO, 3, 0, 1, 1, 0, 100, false).

	onUpdate := func() {
		go t.App.Draw()
	}

	row := 0
	spinner := MakeSpinner()
	spinnerChan := make(chan string)
	pSpinnerChan := make(chan string)
	stopChan := make(chan bool)
	pStopChan := make(chan bool)
	var logLock sync.Mutex
	onLog := func(s string) {
		go func() {
			logLock.Lock()
			pStopChan = stopChan
			stopChan = make(chan bool)
			pSpinnerChan = spinnerChan
			spinnerChan = make(chan string)
			if row != 0 {
				pStopChan <- true
				close(pSpinnerChan)
				logs.SetCellSimple(row-1, 0, "|")
				t.OnUpdate <- true
			}
			logs.SetCellSimple(row, 0, "|")
			logs.SetCellSimple(row, 1, s)
			log.Printf("writing %d %s", row, s)
			idx := row
			row++
			t.OnUpdate <- true
			logLock.Unlock()
			go spinner.Timer(100*time.Millisecond, spinnerChan, stopChan)
			for {
				myS, ok := <-spinnerChan
				if ok {
					logs.SetCell(idx, 0, tview.NewTableCell(myS).SetAlign(tview.AlignRight))
				} else {
					return
				}
				t.OnUpdate <- true
			}
		}()
	}
	var pdfLock sync.Mutex
	pdfRow := 0
	onPdf := func(s string) {
		go func() {
			pdfLock.Lock()
			pdfs.SetCellSimple(pdfRow, 0, s)
			t.OnUpdate <- true
			pdfRow++
			pdfLock.Unlock()
		}()
	}

	var netLock sync.Mutex

	networkUsage := make([]float64, sparklineNetWidth)
	onNet := func(n api.NetData) {
		go func() {
			netLock.Lock()
			s := n.Message
			sl := s[0:min(len(s), 4096)]
			netPage.SetText(sl, false)
			usage := float64(n.Size)
			fastAppend(networkUsage, usage)
			sparkLineNet.SetData(networkUsage)
			t.OnUpdate <- true
			netLock.Unlock()
		}()
	}

	go listen(t, formSubmit, onUpdate, onLog, onPdf, onNet)
	if err := t.App.SetRoot(grid, true).EnableMouse(true).Run(); err != nil {
		t.App.Stop()
		panic(err)
	}
}

func fastAppend(s []float64, v float64) error {
	var i int
	l := len(s)
	if l == 0 {
		return errors.New("Empty slice passed to fastAppend")
	}
	for i = 1; i < l; i++ {
		s[i-1] = s[i]
	}
	s[l-1] = v
	return nil
}
