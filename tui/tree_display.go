package tui

import (
	"log"
	"time"

	"github.com/benjaminchristie/go-arxiv-tree/tree"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TreeDisplay struct {
	ArxivTree  *tree.Node
	TviewTree  *tview.TreeView
	UpdateChan chan bool
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
		t = &TreeDisplay{
			ArxivTree:  head,
			TviewTree:  tview.NewTreeView(),
			UpdateChan: make(chan bool),
		}
		add(root, head)
	} else {
		root := tview.NewTreeNode("").
			SetColor(tcell.ColorRed)
		t = &TreeDisplay{
			ArxivTree:  head,
			TviewTree:  tview.NewTreeView().SetRoot(root).SetCurrentNode(root),
			UpdateChan: make(chan bool),
		}
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
		log.Printf("Adding %v to tree", child)
		hasChildren := len(child.Cites) != 0
		node := tview.NewTreeNode(child.Info.Title).
			SetReference(child).
			SetSelectable(hasChildren)
		if hasChildren {
			node.SetColor(tcell.ColorGreen)
		}
		target.AddChild(node)
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
	if t.TviewTree != nil && t.ArxivTree != nil {
		refresh(t.TviewTree.GetRoot(), t.ArxivTree.Head)
	}
}

func refresh(target *tview.TreeNode, node *tree.Node) {
	if target == nil || node == nil {
		log.Printf("Target or node is nil")
		return
	}
	target = target.ClearChildren()
	add(target, node)
}

func spin(t *TreeDisplay) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		// <-t.UpdateChan
		<-ticker.C
		render(t)
	}
}
