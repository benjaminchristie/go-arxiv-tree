package tui

import (
	"log"
	"sync"

	"github.com/benjaminchristie/go-arxiv-tree/tree"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TreeDisplay struct {
	ArxivTree  *tree.Node
	TviewTree  *tview.TreeView
	UpdateChan chan bool
	Mutex      sync.Mutex
}

type CustomNode struct {
	tview.TreeNode
	ArxivNode *tree.Node
}

type TuiTree struct {
	tview.TreeView
	Head *tree.Node
}

func makeTreeDisplay(head *tree.Node) *TreeDisplay {
	var t *TreeDisplay
	if head != nil {
		root := tview.NewTreeNode(head.Info.Title).
			SetColor(tcell.ColorRed)
		tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
		t = &TreeDisplay{
			ArxivTree:  head,
			TviewTree:  tree,
			UpdateChan: make(chan bool),
		}
		add(root, head)
	} else {
		panic("Invalid head for treeDisplay")
	}
	t.TviewTree.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			return
		}
		children := node.GetChildren()
		if len(children) == 0 {
			n, ok := ref.(*tree.Node)
			if !ok {
				log.Printf("Error casting to *tree.Node")
				return
			}
			add(node, n)
		} else {
			node.SetExpanded(!node.IsExpanded())
		}
	})
	go spin(t)
	return t
}

func add(target *tview.TreeNode, node *tree.Node) {
	if node == nil {
		return
	}
	for _, child := range node.Cites {
		hasChildren := len(child.Cites) != 0
		node := tview.NewTreeNode(child.Info.Title).
			SetReference(child).
			SetSelectable(true)
		target.AddChild(node)
		if hasChildren {
			node.SetColor(tcell.ColorGreen)
		}
	}
}

func updateHead(t *TreeDisplay, head *tree.Node) *TreeDisplay {
	root := tview.NewTreeNode(head.Info.Title).
		SetColor(tcell.ColorRed)
	t.TviewTree = t.TviewTree.SetRoot(root).SetCurrentNode(root)
	t.ArxivTree = head
	add(root, head)
	return t
}

func findNode(t *tree.Node, isTrue func(*tree.Node) bool) *tree.Node {
	if isTrue(t) {
		return t
	}
	var node *tree.Node
	for _, node := range t.Cites {
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

func render(t *TreeDisplay) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	if t.TviewTree != nil {
		refresh(t.TviewTree.GetRoot(), t.ArxivTree)
	}
}

func refresh(target *tview.TreeNode, node *tree.Node) {
	if target == nil {
		log.Printf("Target is nil")
		return
	}
	if node == nil {
		log.Printf("Node is nil")
		return
	}
	target = target.ClearChildren()
	add(target, node)
}

func spin(t *TreeDisplay) {
	for {
		<-t.UpdateChan
		render(t)
	}
}
