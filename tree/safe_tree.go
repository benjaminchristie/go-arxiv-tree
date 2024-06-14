package tree

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	"github.com/jschaf/bibtex"
)

func safeTuiGetInfos(info NodeInfo, netchan chan api.NetData) ([]NodeInfo, error) {
	var entries []bibtex.Entry
	var err error
	if info.BibPath == "" { // bib probably not downloaded
		fh, err := os.CreateTemp("", info.ID)
		if err != nil {
			return nil, err
		}
		filename := fh.Name()
		info.SourcePath = filename
		err = api.SafeTuiDownloadSource(info.ID, filename, netchan)
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
func safeGetInfos(info NodeInfo) ([]NodeInfo, error) {
	var entries []bibtex.Entry
	var err error
	if info.BibPath == "" { // bib probably not downloaded
		fh, err := os.CreateTemp("", info.ID)
		if err != nil {
			return nil, err
		}
		filename := fh.Name()
		info.SourcePath = filename
		err = api.SafeDownloadSource(info.ID, filename)
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

func _safePopulateTree(n *Node, depth int, dolog bool, wg *sync.WaitGroup) {
	if depth <= 0 {
		return
	}
	if dolog {
		au := n.Info.Author
		ti := n.Info.Title
		log.Printf("Populating %d-Tree for %.20s: %.60s", depth, au, ti)
	}
	infos, err := safeGetInfos(n.Info)
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
			_safePopulateTree(c, depth-1, dolog, wg)
			wg.Done()
			<-workerPool
		}(n.Cites[i])
	}
}

func _safeAsyncLoggingPopulateTree(n *Node, depth int, wg *sync.WaitGroup, pdfchan chan string, netchan chan api.NetData, prefix string, cb func(n *Node)) {
	au := n.Info.Author
	ti := n.Info.Title
	pdfStr := fmt.Sprintf("%s%.20s: %.60s", prefix, au, ti)
	log.Printf(pdfStr)
	go cb(n)
	if depth <= 0 {
		return
	}
	infos, err := safeTuiGetInfos(n.Info, netchan)
	pdfchan <- pdfStr
	if err != nil {
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
		go func(node *Node) {
			_safeAsyncLoggingPopulateTree(node, depth-1, wg, pdfchan, netchan, prefix+"-> ", cb)
			wg.Done()
			<-workerPool
		}(n.Cites[i])
	}
}

func SafePopulateTree(n *Node, depth int, dolog bool) {
	var wg sync.WaitGroup
	_safePopulateTree(n, depth, dolog, &wg)
	wg.Wait()
}

func SafeAsyncLoggingPopulateTree(n *Node, depth int, pdfchan chan string, netchan chan api.NetData, cb func(n *Node)) {
	var wg sync.WaitGroup
	_safeAsyncLoggingPopulateTree(n, depth, &wg, pdfchan, netchan, "", cb)
	wg.Wait()
}
