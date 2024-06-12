package main

import (
	"log"

	"github.com/benjaminchristie/go-arxiv-tree/internal/api"
	"github.com/benjaminchristie/go-arxiv-tree/internal/tree"
)

func main() {
	_, err := api.DownloadSource("2404.17906.tar.gz")
	if err != nil {
		panic(err)
	}
	bs, err := tree.ReadBibtexFile("2404.17906/citations.bib")
	if err != nil {
		panic(err)
	}
	t := tree.MakeTree(bs[0])
	tree.PopulateTree(t, 2, true)
	// for _, n := range t.Cites {
	// 	b := n.Entry
	// 	author := b.Tags[bibtex.FieldAuthor].(*ast.UnparsedText)
	// 	title := b.Tags[bibtex.FieldTitle].(*ast.UnparsedText)
	// 	log.Printf("%s: %s\n", author.Value, title.Value)
	// }
	err = tree.Visualize(t, "out.gv")
	log.Printf("%v", err)
	// for _, b := range bs {
	// 	author := b.Tags[bibtex.FieldAuthor].(*ast.UnparsedText)
	// 	title := b.Tags[bibtex.FieldTitle].(*ast.UnparsedText)
	// 	fmt.Printf("%s: %s\n", author.Value, title.Value)
	// }
	// for _, a := range author.Values {
	// 	fmt.Printf("Author: %s ", a)
	// }
	// for _, a := range title.Values {
	// 	fmt.Printf("Title: %s ", a)
	// }
}
