package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	"github.com/benjaminchristie/go-arxiv-tree/tree"
	"github.com/benjaminchristie/go-arxiv-tree/tui"
)

func main() {
	var id string
	var depth int
	var t *tree.Node
	var err error

	tuiPtr := flag.Bool("tui", true, "use tui")
	dirPtr := flag.String("dir", "arxiv-download-folder", "directory to save pdfs to")
	drawPtr := flag.String("viz-out", "", "file to output graph info to")
	auPtr := flag.Bool("author", false, "pass this flag to search by author")
	tiPtr := flag.Bool("title", false, "pass this flag to search by title")
	idPtr := flag.Bool("id", false, "pass this flag to search by id")
	noLogPtr := flag.Bool("silent", false, "pass this flag to disable logging")
	flag.Parse()

	if *tuiPtr {
		tui.Run()
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
			t, err = tree.MakeNodeFromAuthor(id)
			if err != nil {
				log.Fatal(err)
			}
		} else if *tiPtr {
			fmt.Printf("Enter Title to search: ")
			if scanner.Scan() {
				id = scanner.Text()
			}
			fmt.Printf("Enter max tree depth: ")
			fmt.Scanf("%d", &depth)
			log.Printf("searching for %s with depth %d", id, depth)
			t, err = tree.MakeNodeFromTitle(id)
			if err != nil {
				log.Fatal(err)
			}
		} else if *idPtr {
			fmt.Printf("Enter ID to search: ")
			if scanner.Scan() {
				id = scanner.Text()
			}
			fmt.Printf("Enter max tree depth: ")
			fmt.Scanf("%d", &depth)
			log.Printf("searching for %s with depth %d", id, depth)
			t, err = tree.MakeNodeFromID(id)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			fmt.Printf("No flags passed. Defaulting to title search.\n")
			fmt.Printf("Enter title to search: ")
			if scanner.Scan() {
				id = scanner.Text()
			}
			fmt.Printf("Enter max tree depth: ")
			fmt.Scanf("%d", &depth)
			log.Printf("searching for %s with depth %d", id, depth)
			t, err = tree.MakeNodeFromTitle(id)
			if err != nil {
				log.Fatal(err)
			}
		}
		tree.PopulateTree(t, depth, !*noLogPtr)
		err = os.MkdirAll(*dirPtr, 0755)
		if err != nil {
			log.Fatalf("Couldn't create directory %s", *dirPtr)
		}
		tree.Traverse(t, func(n *tree.Node) {
			if n.Info.ID != "" {
				log.Printf("Downloading PDF: %.20s: %.60s", n.Info.Author, n.Info.Title)
				api.DownloadPDF(n.Info.ID, fmt.Sprintf("%s/%s_%s.pdf", *dirPtr, strings.Replace(n.Info.Title, "/", "", -1), n.Info.ID))
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
