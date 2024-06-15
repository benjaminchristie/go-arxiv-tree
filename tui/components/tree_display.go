package components

import (
	log "github.com/benjaminchristie/go-arxiv-tree/arxiv_logger"
	"sync"

	"github.com/benjaminchristie/go-arxiv-tree/tree"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TreeDisplay struct {
	*tview.TreeView
	ArxivHead  *tree.ArxivTree
	mutex      *sync.Mutex
	UpdateChan chan bool
}

func MakeTreeDisplay(head *tree.ArxivTree) *TreeDisplay {
	var m sync.Mutex
	t := &TreeDisplay{
		TreeView:   tview.NewTreeView(),
		UpdateChan: make(chan bool),
		mutex:      &m,
	}
	t.TreeView.SetBorder(true).SetBorderColor(tcell.ColorOrangeRed).SetTitle("Current Tree")
	if head != nil {
		t.UpdateHead(head)
	}
	t.SetSelectedFunc(
		func(node *tview.TreeNode) {
			ref := node.GetReference()
			if ref == nil {
				return
			}
			children := node.GetChildren()
			if len(children) == 0 {
				n, ok := ref.(*tree.ArxivTree)
				if !ok {
					log.Printf("Error casting to *tree.ArxivTree")
				}
				addNode(node, n)
			} else {
				node.SetExpanded(!node.IsExpanded())
			}
		},
	)
	go t.spin()
	return t
}

func (t *TreeDisplay) render() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.ArxivHead == nil {
		log.Printf("t.ArxivHead is nil")
		return
	}
	target := t.GetRoot()
	if target == nil {
		log.Printf("target is nil")
		return
	}
	target.ClearChildren()
	addNode(target, t.ArxivHead)

}

func (t *TreeDisplay) spin() {
	for {
		_, ok := <-t.UpdateChan
		if ok {
			t.render()
		}
	}
}

func (t *TreeDisplay) UpdateHead(head *tree.ArxivTree) {
	root := tview.NewTreeNode(head.Value.(tree.ArxivTreeInfo).Title).
		SetColor(tcell.ColorRed)
	t.SetRoot(root).SetCurrentNode(root)
	t.ArxivHead = head
	addNode(root, head)
}

func addNode(target *tview.TreeNode, node *tree.ArxivTree) {
	if node == nil {
		return
	}
	for _, child := range node.Children {
		hasChildren := len(child.Children) != 0
		node := tview.NewTreeNode(child.Value.(tree.ArxivTreeInfo).Title).
			SetReference(child).
			SetSelectable(true)
		target.AddChild(node)
		if hasChildren {
			node.SetColor(tcell.ColorGreen)
		}
	}
}

func findNode(t *tree.ArxivTree, isTrue func(*tree.ArxivTree) bool) *tree.ArxivTree {
	if isTrue(t) {
		return t
	}
	var node *tree.ArxivTree
	for _, node := range t.Children {
		if isTrue(node) {
			return node
		}
		node = findNode(node, isTrue)
		if node != nil {
			return node
		}
	}
	return node
}
