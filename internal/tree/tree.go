package tree

import (
	"os"

	"github.com/jschaf/bibtex"
)

type Node struct {
	Head  *Node
	Entry bibtex.Entry
	Cites []*Node
}

var biber *bibtex.Biber

func init() {
	biber = bibtex.New()
}

func getCitations(_ bibtex.Entry) []*bibtex.Entry {
	return nil
}

func readBibtexFile() ([]bibtex.Entry, error) {
	f, err := os.Open("refs.bib")
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
