package tree

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/benjaminchristie/go-arxiv-tree/internal/api"
	"github.com/benjaminchristie/go-arxiv-tree/internal/parser"
	"github.com/jschaf/bibtex"
)

type Node struct {
	Head  *Node
	Entry bibtex.Entry
	Cites []*Node
}

var biber *bibtex.Biber
var workerPool chan bool

func init() {
	biber = &bibtex.Biber{}
	N := runtime.GOMAXPROCS(0)
	workerPool = make(chan bool, N)
	// for range N {
	// 	workerPool <- true
	// }
}

func getCitations(e bibtex.Entry) ([]bibtex.Entry, error) {
	_, title, err := api.QueryBibtexEntry(e)
	if err != nil {
		return nil, err
	}
	p := api.QueryRequest{
		SearchQuery: fmt.Sprintf("ti:%s", title),
	}
	x, err := api.Query("query", p)
	if err != nil {
		return nil, err
	}
	s := parser.ParseXML(x)
	if len(s) == 0 {
		return nil, errors.New("Empty XML")
	}
	true_id := s[0].Id[strings.LastIndex(s[0].Id, "/")+1:]
	archiveFile, err := api.DownloadSource(true_id + ".tar.gz")
	if err != nil {
		return nil, err
	}
	idx := strings.LastIndex(archiveFile, "/")
	if idx <= 0 {
		return nil, errors.New("Invalid archiveFile path")
	}
	dir := archiveFile[0:idx]
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var f fs.DirEntry
	found := false
	for _, f = range files {
		if f.IsDir() {
			continue
		}
		fn := f.Name()
		fn_len := len(fn)
		if fn[fn_len-4:fn_len] == ".bib" {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("No .bib file found")
	}
	return ReadBibtexFile(fmt.Sprintf("%s/%s", dir, f.Name()))
}

func ReadBibtexFile(filename string) ([]bibtex.Entry, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	astfptr, err := biber.Parse(f)
	if err != nil {
		return nil, err
	}
	entries, err := biber.Resolve(astfptr)
	return entries, err
}

func MakeTree(e ...bibtex.Entry) *Node {
	var n *Node
	if len(e) > 0 {
		n = &Node{
			Head:  nil,
			Entry: e[0],
			Cites: nil,
		}
	} else {
		n = &Node{
			Head:  nil,
			Cites: nil,
		}
	}
	return n
}

func entry(n *Node, depth int) {
	var wg sync.WaitGroup
	if depth <= 0 {
		return
	}
	helper := func(node *Node) {
		entries, err := getCitations(node.Entry)
		if err != nil {
			return
		}
		node.Cites = make([]*Node, len(entries), len(entries))
		for i, e := range entries {
			node.Cites[i] = &Node{
				Entry: e,
				Head:  node,
				Cites: nil,
			}
		}
	}
	helper(n)
	for _, child := range n.Cites {
		wg.Add(1)
		go func(c *Node) {
			helper(c)
			wg.Done()
		}(child)
	}
	wg.Wait()
}

func helper(node *Node) {
	entries, err := getCitations(node.Entry)
	if err != nil {
		return
	}
	node.Cites = make([]*Node, len(entries), len(entries))
	for i, e := range entries {
		node.Cites[i] = &Node{
			Entry: e,
			Head:  node,
			Cites: nil,
		}
	}
}

func PopulateTree(n *Node, depth int, doLog bool) {
	var wg sync.WaitGroup
	recPopulateTree(n, depth, doLog, &wg)
	wg.Wait()
}

func recPopulateTree(n *Node, depth int, doLog bool, wg *sync.WaitGroup) {
	if depth <= 0 {
		return
	}
	if doLog {
		a, t, _ := api.QueryBibtexEntry(n.Entry)
		log.Printf("Populating %d-Tree for %.20s: %.40s", depth, a, t)
	}
	helper(n)
	for _, child := range n.Cites {
		wg.Add(1)
		workerPool<-true
		go func(c *Node) {
			recPopulateTree(c, depth-1, doLog, wg)
			<-workerPool
			wg.Done()
		}(child)
	}
}

func TraverseDownloadPDF(node *Node, outdir string) {
	var wg sync.WaitGroup
	os.MkdirAll(outdir, 0755)
	f := func(n *Node) error {
		if n == nil {
			return nil
		}
		_, t, _ := api.QueryBibtexEntry(n.Entry)
		p := api.QueryRequest{
			SearchQuery: fmt.Sprintf("ti:%s", t),
		}
		x, err := api.Query("query", p)
		if err != nil {
			return err
		}
		s := parser.ParseXML(x)
		if len(s) == 0 {
			return errors.New("Empty XML")
		}
		true_id := s[0].Id[strings.LastIndex(s[0].Id, "/")+1:]
		log.Printf("Downloading %s", true_id)
		err = api.DownloadPDF(true_id+".tar.gz", fmt.Sprintf("%s/%s.pdf", outdir, t))
		return err
	}
	var g func(n *Node)
	g = func(n *Node) {
		for _, child := range n.Cites {
			wg.Add(2)
			workerPool<-true
			go func(c *Node) {
				f(c)
				wg.Done()
				g(c)
				wg.Done()
				<-workerPool
			}(child)
		}
	}
	f(node)
	g(node)
	wg.Wait()
}
