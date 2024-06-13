# go-arxiv-tree

Recursively downloads the bibliography of (and the bibliography of ...) a paper
listed on arXiv, can also download the associated PDFs of the respective documents.

Execute `go run cmd/main.go -h` for usage, and `dot -Tsvg <output>.gv -o
<output>.svg` to generate an SVG image of the recursive bibliography graph.

PRs are welcome.
