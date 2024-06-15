package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	log "github.com/benjaminchristie/go-arxiv-tree/arxiv_logger"
	"github.com/benjaminchristie/go-arxiv-tree/tree"
	"github.com/benjaminchristie/go-arxiv-tree/tui"
)

func main() {
	var id string
	var depth int
	var t *tree.ArxivTree
	var info *tree.ArxivTreeInfo
	var p api.QueryRequest
	var err error

	tuiPtr := flag.Bool("tui", true, "use tui")
	dirPtr := flag.String("dir", "arxiv-download-folder", "directory to save pdfs to")
	drawPtr := flag.String("viz-out", "", "file to output graph info to")
	auPtr := flag.Bool("author", false, "pass this flag to search by author")
	tiPtr := flag.Bool("title", false, "pass this flag to search by title")
	idPtr := flag.Bool("id", false, "pass this flag to search by id")
	flag.Parse()

	log.Initialize(true, "log.log")

	if *tuiPtr {
		t := tui.MakeTUI()
		t.Run()
	} else {

		scanner := bufio.NewScanner(os.Stdin)

		fmt.Printf("--------------------------------------------------------------------\n")
		fmt.Printf("Welcome to arxiv-tree. Begin your search below. Do not use colon [:]\n" +
			"or the arXiv API will reject your query. Pass -h for help info.\n")
		fmt.Printf("--------------------------------------------------------------------\n")

		if *auPtr {
			fmt.Printf("Enter Author to search: ")
			if scanner.Scan() {
				id = scanner.Text()
			}
			fmt.Printf("Enter max tree depth: ")
			fmt.Scanf("%d", &depth)
			log.Printf("searching for %s with depth %d", id, depth)

			p.Author = id

		} else if *tiPtr {
			fmt.Printf("Enter Title to search: ")
			if scanner.Scan() {
				id = scanner.Text()
			}
			fmt.Printf("Enter max tree depth: ")
			fmt.Scanf("%d", &depth)
			log.Printf("searching for %s with depth %d", id, depth)

			p.Title = id

		} else if *idPtr {
			fmt.Printf("Enter ID to search: ")
			if scanner.Scan() {
				id = scanner.Text()
			}
			fmt.Printf("Enter max tree depth: ")
			fmt.Scanf("%d", &depth)
			log.Printf("searching for %s with depth %d", id, depth)

			p.IDList = id

		} else {
			fmt.Printf("No flags passed. Defaulting to title search.\n")
			fmt.Printf("Enter title to search: ")
			if scanner.Scan() {
				id = scanner.Text()
			}
			fmt.Printf("Enter max tree depth: ")
			fmt.Scanf("%d", &depth)
			log.Printf("searching for %s with depth %d", id, depth)

			p.Title = id

		}
		err = tree.MakeInfoFromQuery(info, p, true)
		if err != nil {
			log.Fatal(err)
		}
		tree.PopulateTree(t, depth, func(at *tree.ArxivTree) {})
		err = os.MkdirAll(*dirPtr, 0755)
		if err != nil {
			log.Fatalf("Couldn't create directory %s", *dirPtr)
		}
		tree.Traverse(t, func(n *tree.ArxivTree) {
			v := n.Value.(tree.ArxivTreeInfo)
			if v.ID != "" {
				log.Printf("Downloading PDF: %.20s: %.60s", v.Author, v.Title)
				api.DownloadPDF(v.ID, fmt.Sprintf("%s/%s_%s.pdf", *dirPtr, strings.Replace(v.Title, "/", "", -1), v.ID))
			} else {
				log.Printf("Could not download PDF, n.Info.ID is empty")
			}
		})

		if *drawPtr != "" {
			log.Printf("Outputing graph view to %s. Run `dot -Tsvg %s -o <file>` to view.", *drawPtr, *drawPtr)
			tree.Visualize(t, *drawPtr)
		}
	}
}
