package tree

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	log "github.com/benjaminchristie/go-arxiv-tree/arxiv_logger"
	"github.com/benjaminchristie/go-arxiv-tree/comms"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/jschaf/bibtex"
)

type ArxivTree struct {
	Head     *ArxivTree
	Value    any
	Children []*ArxivTree
}

type ArxivTreeInfo struct {
	Entry      bibtex.Entry
	Author     string
	ID         string
	SourcePath string
	BibPath    string
	Title      string
}

var biber *bibtex.Biber
var workerPool chan bool

func init() {
	biber = &bibtex.Biber{}
	N := 4 * runtime.GOMAXPROCS(0)
	workerPool = make(chan bool, N)
}

// note that if ID is passed, the xml does not need to be retrieved
// this is a TODO
func MakeInfoFromQuery(info *ArxivTreeInfo, p api.QueryRequest, downloadSource bool, comms ...comms.Comm) error {
	var x string
	var err error
	x, err = api.Query(p)
	if err != nil {
		return err
	}
	for _, c := range comms {
		go c.Send(api.NetData{
			Message: x,
			Size:    len(x),
		})
	}
	xmlEntry := api.ParseXML(x)
	if len(xmlEntry) == 0 {
		return errors.New("Parsing XML Failed: you are probably temporarily banned")
	}
	fid := xmlEntry[0].ID
	id := fid[strings.LastIndex(fid, "/")+1:]
	info.ID = id
	info.Title = xmlEntry[0].Title
	info.Author = xmlEntry[0].Author[0].Name
	var fh *os.File
	fh, err = os.CreateTemp("", id)
	if err != nil {
		return err
	}
	filename := fh.Name()
	info.SourcePath = filename
	err = api.DownloadSource(id, filename, comms...)
	if err != nil {
		return err
	}
	var dirname string
	dirname, err = os.MkdirTemp("", id)
	if err != nil {
		return err
	}
	err = api.ExtractTargz(filename, dirname)
	if err != nil {
		return err
	}
	found := false
	citeFilename := ""
	filepath.WalkDir(dirname, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		fn_len := len(path)
		if path[fn_len-4:fn_len] == ".bib" {
			_, err = api.ReadBibtexFile(path)
			if err != nil {
				return nil
			}
			found = true
			citeFilename = path
			return filepath.SkipAll
		}
		return nil
	})
	if found {
		info.BibPath = citeFilename
	}
	return nil
}

func MakeInfo(info *ArxivTreeInfo, downloadSource bool, comms ...comms.Comm) error {
	var err error
	if info.ID == "" && info.Author == "" && info.Title == "" {
		info.Author, info.Title, err = api.QueryBibtexEntry(info.Entry)
		if err != nil {
			return err
		}
		p := api.QueryRequest{
			Title: info.Title,
		}
		var x string
		x, err = api.Query(p)
		if err != nil {
			return err
		}
		for _, c := range comms {
			go c.Send(x) // c.Send blocks
		}
		xml := api.ParseXML(x)
		if len(xml) == 0 {
			return errors.New("Parsing XML Failed: you are probably temporarily banned")
		}
		fid := xml[0].ID
		id := fid[strings.LastIndex(fid, "/")+1:]
		info.ID = id
	}
	if downloadSource {
		fh, err := os.CreateTemp("", info.ID)
		if err != nil {
			return err
		}
		filename := fh.Name()
		info.SourcePath = filename
		err = api.DownloadSource(info.ID, filename, comms...)
		if err != nil {
			return err
		}
		dirname, err := os.MkdirTemp("", info.ID)
		if err != nil {
			return err
		}
		err = api.ExtractTargz(filename, dirname, comms...)
		if err != nil {
			return err
		}
		found := false
		citeFilename := ""
		filepath.WalkDir(dirname, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			fn_len := len(path)
			if path[fn_len-4:fn_len] == ".bib" {
				_, err = api.ReadBibtexFile(path)
				if err != nil {
					return nil
				}
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
	return nil
}

func MakeTree(e bibtex.Entry, downloadSource bool, id, author, title string) (*ArxivTree, error) {
	info := ArxivTreeInfo{
		Entry:  e,
		ID:     id,
		Author: author,
		Title:  title,
	}
	err := MakeInfo(&info, downloadSource)
	if err != nil {
		return nil, err
	}
	tree := &ArxivTree{
		Head:     nil,
		Value:    info,
		Children: nil,
	}
	return tree, nil
}

func getInfos(info ArxivTreeInfo, comms ...comms.Comm) ([]ArxivTreeInfo, error) {
	var entries []bibtex.Entry
	var err error
	if info.BibPath == "" { // bib probably not downloaded
		fh, err := os.CreateTemp("", info.ID)
		if err != nil {
			log.Printf("error %s", err.Error())
			return nil, err
		}
		filename := fh.Name()
		info.SourcePath = filename
		err = api.DownloadSource(info.ID, filename, comms...)
		if err != nil {
			log.Printf("error %s", err.Error())
			return nil, err
		}
		dirname, err := os.MkdirTemp("", info.ID)
		if err != nil {
			log.Printf("error %s", err.Error())
			return nil, err
		}
		err = api.ExtractTargz(filename, dirname, comms...)
		if err != nil {
			log.Printf("error %s", err.Error())
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
	infos := make([]ArxivTreeInfo, len(entries))
	for i, e := range entries {
		infos[i].Entry = e
		MakeInfo(&infos[i], false, comms...)
	}
	return infos, nil
}

func contains(v *[]*ArxivTree, c *ArxivTree) bool {
	for _, e := range *v {
		if c == e {
			return true
		}
	}
	return false
}

func Traverse(n *ArxivTree, cb func(*ArxivTree)) {
	cb(n)
	for _, c := range n.Children {
		Traverse(c, cb)
	}
}

func Visualize(n *ArxivTree, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	if n == nil {
		return nil
	}
	g := graph.New(graph.StringHash, graph.Directed())
	cb := func(c *ArxivTree) {
		if c == nil {
			return
		}
		t := c.Value.(ArxivTreeInfo).Title
		h_t := c.Head.Value.(ArxivTreeInfo).Title
		g.AddVertex(t)
		g.AddEdge(h_t, t)
	}
	t := n.Value.(ArxivTreeInfo).Title
	g.AddVertex(t)
	Traverse(n, cb)
	err = draw.DOT(g, file)
	return err
}

func _populateTree(t *ArxivTree, depth int, wg *sync.WaitGroup, prefix string, cb func(*ArxivTree), comms ...comms.Comm) {
	go func() {
		wg.Add(1)
		cb(t)
		wg.Done()
	}()
	if depth <= 0 {
		log.Printf("Reached search depth at %s", t.Value.(ArxivTreeInfo).Title)
		return
	}
	infos, err := getInfos(t.Value.(ArxivTreeInfo), comms...)
	if err != nil {
		log.Printf("Error in getInfos: %s", err.Error())
		return
	}
	t.Children = make([]*ArxivTree, len(infos))
	for i, info := range infos {
		t.Children[i] = &ArxivTree{
			Head:     t,
			Value:    info,
			Children: nil,
		}
		workerPool <- true
		wg.Add(1)
		go func(n *ArxivTree) {
			_populateTree(n, depth-1, wg, prefix, cb, comms...)
			wg.Done()
			<-workerPool
		}(t.Children[i])
	}
}

func PopulateTree(t *ArxivTree, depth int, cb func(*ArxivTree), comms ...comms.Comm) {
	var wg sync.WaitGroup
	_populateTree(t, depth, &wg, "", cb, comms...)
	wg.Wait()

}
