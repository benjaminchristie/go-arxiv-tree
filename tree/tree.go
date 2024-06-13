package tree

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/jschaf/bibtex"
)

func getInfos(info NodeInfo) ([]NodeInfo, error) {
	var entries []bibtex.Entry
	var err error
	if info.BibPath == "" { // bib probably not downloaded
		fh, err := os.CreateTemp("", info.ID)
		if err != nil {
			return nil, err
		}
		filename := fh.Name()
		info.SourcePath = filename
		err = api.DownloadSource(info.ID, filename)
		if err != nil {
			return nil, err
		}
		dirname, err := os.MkdirTemp("", info.ID)
		if err != nil {
			return nil, err
		}
		err = api.ExtractTargz(filename, dirname)
		if err != nil {
			return nil, err
		}
		found := false
		citeFilename := ""
		filepath.WalkDir(dirname, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			fn_len := len(path)
			if path[fn_len-4:fn_len] == ".bib" {
				found = true
				citeFilename = path
				return filepath.SkipAll
			}
			return nil
		})
		if found {
			info.BibPath = citeFilename
		}
	}
	entries, err = api.ReadBibtexFile(info.BibPath)
	if err != nil {
		return nil, err
	}
	infos := make([]NodeInfo, len(entries))
	for i, e := range entries {
		infos[i], _ = MakeNodeInfo(e, false)
	}
	return infos, nil
}

func _populateTree(n *Node, depth int, dolog bool, wg *sync.WaitGroup) {
	if depth <= 0 {
		return
	}
	if dolog {
		au := n.Info.Author
		ti := n.Info.Title
		log.Printf("Populating %d-Tree for %.20s: %.60s", depth, au, ti)
	}
	infos, err := getInfos(n.Info)
	if err != nil {
		log.Printf("error in _populateTree: %v %v", n.Info, err)
		return
	}
	n.Cites = make([]*Node, len(infos))
	for i, info := range infos {
		n.Cites[i] = &Node{
			Head:  n,
			Info:  info,
			Cites: nil,
		}
		workerPool <- true
		wg.Add(1)
		go func(c *Node) {
			_populateTree(c, depth-1, dolog, wg)
			wg.Done()
			<-workerPool
		}(n.Cites[i])
	}
}

func PopulateTree(n *Node, depth int, dolog bool) {
	var wg sync.WaitGroup
	_populateTree(n, depth, dolog, &wg)
	wg.Wait()
}

func Traverse(n *Node, cb func(*Node)) {
	cb(n)
	for _, c := range n.Cites {
		Traverse(c, cb)
	}
}

func Visualize(n *Node, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	g := graph.New(graph.StringHash, graph.Directed())
	cb := func(c *Node) {
		t := c.Info.Title
		h_t := c.Head.Info.Title
		g.AddVertex(t)
		g.AddEdge(h_t, t)
	}
	t := n.Info.Title
	g.AddVertex(t)
	g.AddVertex(t)
	Traverse(n, cb)
	err = draw.DOT(g, file)
	return err
}

func contains(v *[]*Node, c *Node) bool {
	for _, e := range *v {
		if c == e {
			return true
		}
	}
	return false
}
