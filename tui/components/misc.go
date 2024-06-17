package components

import (
	"errors"
	"sync"
	"time"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	log "github.com/benjaminchristie/go-arxiv-tree/arxiv_logger"
	"github.com/benjaminchristie/go-arxiv-tree/comms"
	"github.com/benjaminchristie/go-arxiv-tree/tree"
	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
)

func AddToGrid(g *tview.Grid, t *TUIPrimitive) {
	p := t.GridParameters
	g.AddItem(
		t.Primitive,
		p.row,
		p.column,
		p.rowSpan,
		p.colSpan,
		p.minGridHeight,
		p.minGridWidth,
		p.focus,
	)
}

func MakeLogs(c *comms.Comm) *TUIPrimitive {
	logs := tview.NewTable()
	logs.SetBorder(true).
		SetTitle("Logs").
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitleColor(tcell.ColorGhostWhite).
		SetTitleAlign(tview.AlignCenter)

	var logLock sync.Mutex
	row := 0
	spinner := MakeSpinner()
	spinnerChan := make(chan string)
	pSpinnerChan := make(chan string)
	stopChan := make(chan bool)
	pStopChan := make(chan bool)
	go func() {
		for {
			pS, ok := <-c.PublicChan
			if !ok {
				log.Printf("PublicChan closed")
				continue
			}
			s, ok := pS.(string)
			if !ok {
				log.Printf("Error casting to string in MakeLogs callback")
				continue
			}
			logLock.Lock()

			pStopChan = stopChan
			stopChan = make(chan bool)
			pSpinnerChan = spinnerChan
			spinnerChan = make(chan string)

			if row != 0 {
				pStopChan <- true
				close(pSpinnerChan)
				logs.SetCellSimple(row-1, 0, "|")
			}
			logs.SetCellSimple(row, 0, "|")
			logs.SetCellSimple(row, 1, s)
			logLock.Unlock()
			go spinner.Timer(100*time.Millisecond, spinnerChan, stopChan)
			go func(i int) {
				for {
					myS, ok := <-spinnerChan
					if !ok {
						return
					}
					logs.SetCell(i, 0, tview.NewTableCell(myS).SetAlign(tview.AlignRight))
				}
			}(row)
			row++
		}
	}()

	return &TUIPrimitive{
		Primitive: logs,
		GridParameters: GridParam{
			row:           2,
			column:        2,
			rowSpan:       1,
			colSpan:       2,
			minGridHeight: 0,
			minGridWidth:  100,
			focus:         false,
		},
	}
}

func MakePDFLogs(c *comms.Comm) *TUIPrimitive {
	logs := tview.NewTable()
	logs.SetBorder(true).
		SetTitle("PDFs").
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitleColor(tcell.ColorGhostWhite).
		SetTitleAlign(tview.AlignCenter)

	var pdfLock sync.Mutex
	row := 0
	go func() {
		for {
			pS, ok := <-c.PublicChan
			if !ok {
				log.Printf("PublicChan closed")
				continue
			}
			s, ok := pS.(string)
			if !ok {
				log.Printf("Error casting to string in MakePDFLogs callback")
				continue
			}
			pdfLock.Lock()
			logs.SetCellSimple(row, 0, s)
			row++
			pdfLock.Unlock()
		}
	}()
	return &TUIPrimitive{
		Primitive: logs,
		GridParameters: GridParam{
			row:           3,
			column:        2,
			rowSpan:       1,
			colSpan:       2,
			minGridHeight: 0,
			minGridWidth:  100,
			focus:         false,
		},
	}
}

func makeNetPage() *TUIPrimitive {
	netPage := tview.NewTextArea()
	netPage.SetBorder(true).
		SetTitle("Network").
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitleColor(tcell.ColorGhostWhite).
		SetTitleAlign(tview.AlignCenter)

	return &TUIPrimitive{
		Primitive: netPage,
		GridParameters: GridParam{
			row:           2,
			column:        0,
			rowSpan:       1,
			colSpan:       2,
			minGridHeight: 0,
			minGridWidth:  100,
			focus:         false,
		},
	}
}

func makeSparkline() *TUIPrimitive {
	sparkLineNet := tvxwidgets.NewSparkline()
	sparkLineNet.SetBorder(true).
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitle("Network IO")
	return &TUIPrimitive{
		Primitive: sparkLineNet,
		GridParameters: GridParam{
			row:           3,
			column:        0,
			rowSpan:       1,
			colSpan:       2,
			minGridHeight: 0,
			minGridWidth:  100,
			focus:         false,
		},
	}
}

func MakeNet(c *comms.Comm) (*TUIPrimitive, *TUIPrimitive) {
	netpage := makeNetPage()
	sparkline := makeSparkline()

	_, _, sparklineNetWidth, _ := sparkline.Primitive.(*tvxwidgets.Sparkline).
		GetInnerRect()

	if sparklineNetWidth == 0 {
		sparklineNetWidth = 15
	}

	networkUsage := make([]float64, sparklineNetWidth*5)
	var lock sync.Mutex
	go func() {
		log.Printf("In make net go")
		for {
			pS, ok := <-c.PublicChan
			if !ok {
				log.Printf("PublicChan closed")
				continue
			}
			m, ok := pS.(api.NetData)
			if !ok {
				log.Printf("Error casting to api.NetData in MakeNet callback")
				continue
			}
			lock.Lock()
			s := m.Message
			netpage.Primitive.(*tview.TextArea).SetText(s, false)
			usage := float64(m.Size)
			fastAppend(networkUsage, usage)
			sparkline.Primitive.(*tvxwidgets.Sparkline).SetData(networkUsage)
			lock.Unlock()
		}
	}()
	return netpage, sparkline
}

func MakeTreeDisplayComponent(t *tree.ArxivTree, c *comms.Comm) *TUIPrimitive {
	p := MakeTreeDisplay(t)

	go func() {
		for {
			v, ok := <-c.PublicChan
			log.Printf("in go func: %v", v)
			if !ok {
				log.Printf("PublicChan closed")
				continue
			}
			p.UpdateChan <- v.(bool)
		}
	}()
	return &TUIPrimitive{
		Primitive: p,
		GridParameters: GridParam{
			row:           0,
			column:        2,
			rowSpan:       2,
			colSpan:       2,
			minGridHeight: 0,
			minGridWidth:  100,
			focus:         false,
		},
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
