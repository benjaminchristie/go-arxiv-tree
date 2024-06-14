package tree

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/benjaminchristie/go-arxiv-tree/api"
	"github.com/jschaf/bibtex"
)

// some data contained in this struct is redundant, but it is useful to store
// it here
type NodeInfo struct {
	Entry      bibtex.Entry
	Author     string
	ID         string
	SourcePath string
	BibPath    string
	Title      string
}

type Node struct {
	Head  *Node
	Info  NodeInfo
	Cites []*Node
}

var biber *bibtex.Biber
var workerPool chan bool

func init() {
	biber = &bibtex.Biber{}
	N := 4 * runtime.GOMAXPROCS(0)
	workerPool = make(chan bool, N)
}

// id, author, title
func MakeNodeInfo(e bibtex.Entry, downloadSource bool, s ...string) (NodeInfo, error) {
	var err error
	info := NodeInfo{}
	if len(s) == 0 {
		var a, t string
		a, t, err = api.QueryBibtexEntry(e)
		if err != nil {
			return info, err
		}
		info.Author = a
		info.Title = t
		p := api.QueryRequest{
			Title: t,
		}
		var x string
		x, err = api.Query(p)
		if err != nil {
			return info, err
		}
		xmlEntry := api.ParseXML(x)
		if len(xmlEntry) == 0 {
			return info, errors.New("Parsing XML Failed")
		}
		fid := xmlEntry[0].ID
		id := fid[strings.LastIndex(fid, "/")+1:]
		info.ID = id
	} else {
		if len(s) >= 1 {
			info.ID = s[0]
		}
		if len(s) >= 2 {
			info.Author = s[1]
		}
		if len(s) >= 3 {
			info.Title = s[2]
		}
	}
	if downloadSource {
		fh, err := os.CreateTemp("", info.ID)
		if err != nil {
			return info, err
		}
		filename := fh.Name()
		info.SourcePath = filename
		err = api.DownloadSource(info.ID, filename)
		if err != nil {
			return info, err
		}
		dirname, err := os.MkdirTemp("", info.ID)
		if err != nil {
			return info, err
		}
		err = api.ExtractTargz(filename, dirname)
		if err != nil {
			return info, err
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
	return info, nil
}

// id, author, title
func MakeNode(e bibtex.Entry, downloadSource bool, s ...string) (*Node, error) {
	info, err := MakeNodeInfo(e, downloadSource, s...)
	if err != nil {
		return nil, err
	}
	node := &Node{
		Head:  nil,
		Info:  info,
		Cites: nil,
	}
	return node, err
}
func tuiMakeNodeFromXHelper(node *Node, p api.QueryRequest, netchan chan api.NetData) (*Node, error) {
	var x string
	var err error
	x, err = api.Query(p)
	if err != nil {
		return node, err
	}
	netchan <- api.NetData{
		Message: x,
		Size:    len(x),
	}
	xmlEntry := api.ParseXML(x)
	if len(xmlEntry) == 0 {
		return node, errors.New("Parsing XML Failed")
	}
	fid := xmlEntry[0].ID
	id := fid[strings.LastIndex(fid, "/")+1:]
	node.Info.ID = id
	node.Info.Title = xmlEntry[0].Title
	node.Info.Author = xmlEntry[0].Author[0].Name
	fh, err := os.CreateTemp("", node.Info.ID)
	if err != nil {
		return node, err
	}
	filename := fh.Name()
	node.Info.SourcePath = filename
	err = api.TuiDownloadSource(node.Info.ID, filename, netchan)
	if err != nil {
		return node, err
	}
	dirname, err := os.MkdirTemp("", node.Info.ID)
	if err != nil {
		return node, err
	}
	err = api.ExtractTargz(filename, dirname)
	if err != nil {
		return node, err
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
		node.Info.BibPath = citeFilename
	}
	return node, nil
}
func makeNodeFromXHelper(node *Node, p api.QueryRequest) (*Node, error) {
	var x string
	var err error
	x, err = api.Query(p)
	if err != nil {
		return node, err
	}
	xmlEntry := api.ParseXML(x)
	if len(xmlEntry) == 0 {
		return node, errors.New("Parsing XML Failed")
	}
	fid := xmlEntry[0].ID
	id := fid[strings.LastIndex(fid, "/")+1:]
	node.Info.ID = id
	node.Info.Title = xmlEntry[0].Title
	node.Info.Author = xmlEntry[0].Author[0].Name
	fh, err := os.CreateTemp("", node.Info.ID)
	if err != nil {
		return node, err
	}
	filename := fh.Name()
	node.Info.SourcePath = filename
	err = api.DownloadSource(node.Info.ID, filename)
	if err != nil {
		return node, err
	}
	dirname, err := os.MkdirTemp("", node.Info.ID)
	if err != nil {
		return node, err
	}
	err = api.ExtractTargz(filename, dirname)
	if err != nil {
		return node, err
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
		node.Info.BibPath = citeFilename
	}
	return node, nil
}

// should probably only be used when initializing tree in main
func MakeNodeFromID(id string) (*Node, error) {
	info := NodeInfo{
		Entry:      bibtex.Entry{},
		Author:     "",
		ID:         "",
		SourcePath: "",
		BibPath:    "",
		Title:      "",
	}
	node := &Node{
		Head:  nil,
		Info:  info,
		Cites: nil,
	}
	p := api.QueryRequest{
		IDList: id,
	}
	return makeNodeFromXHelper(node, p)
}

func MakeNodeFromAuthor(id string) (*Node, error) {
	info := NodeInfo{
		Entry:      bibtex.Entry{},
		Author:     "",
		ID:         "",
		SourcePath: "",
		BibPath:    "",
		Title:      "",
	}
	node := &Node{
		Head:  nil,
		Info:  info,
		Cites: nil,
	}
	p := api.QueryRequest{
		Author: id,
	}
	return makeNodeFromXHelper(node, p)
}

func MakeNodeFromTitle(id string) (*Node, error) {
	info := NodeInfo{
		Entry:      bibtex.Entry{},
		Author:     "",
		ID:         "",
		SourcePath: "",
		BibPath:    "",
		Title:      "",
	}
	node := &Node{
		Head:  nil,
		Info:  info,
		Cites: nil,
	}
	p := api.QueryRequest{
		Title: id,
	}
	return makeNodeFromXHelper(node, p)
}
func TuiMakeNodeFromID(id string, netchan chan api.NetData) (*Node, error) {
	info := NodeInfo{
		Entry:      bibtex.Entry{},
		Author:     "",
		ID:         "",
		SourcePath: "",
		BibPath:    "",
		Title:      "",
	}
	node := &Node{
		Head:  nil,
		Info:  info,
		Cites: nil,
	}
	p := api.QueryRequest{
		IDList: id,
	}
	return tuiMakeNodeFromXHelper(node, p, netchan)
}

func TuiMakeNodeFromAuthor(id string, netchan chan api.NetData) (*Node, error) {
	info := NodeInfo{
		Entry:      bibtex.Entry{},
		Author:     "",
		ID:         "",
		SourcePath: "",
		BibPath:    "",
		Title:      "",
	}
	node := &Node{
		Head:  nil,
		Info:  info,
		Cites: nil,
	}
	p := api.QueryRequest{
		Author: id,
	}
	return tuiMakeNodeFromXHelper(node, p, netchan)
}

func TuiMakeNodeFromTitle(id string, netchan chan api.NetData) (*Node, error) {
	info := NodeInfo{
		Entry:      bibtex.Entry{},
		Author:     "",
		ID:         "",
		SourcePath: "",
		BibPath:    "",
		Title:      "",
	}
	node := &Node{
		Head:  nil,
		Info:  info,
		Cites: nil,
	}
	p := api.QueryRequest{
		Title: id,
	}
	return tuiMakeNodeFromXHelper(node, p, netchan)
}
