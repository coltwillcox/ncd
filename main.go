package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TODO Add theming

func main() {
	result := make(chan string, 1)

	app := tview.NewApplication()
	headerRow1 := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetTextColor(tcell.ColorWhiteSmoke).SetText("Neon Change Directory")
	headerRow1.SetBackgroundColor(tcell.ColorRoyalBlue)
	headerRow2 := tview.NewTextView()
	headerRow2.SetBackgroundColor(tcell.ColorWhiteSmoke)
	footerRow1 := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetTextColor(tcell.ColorWhiteSmoke)
	footerRow1.SetBackgroundColor(tcell.ColorWhiteSmoke)
	footerRow2a := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetTextColor(tcell.ColorWhiteSmoke).SetText("Speed search:")
	footerRow2a.SetBackgroundColor(tcell.ColorRoyalBlue).SetBorderPadding(0, 0, 1, 1)
	footerRow2b := tview.NewInputField().SetFieldWidth(20)
	footerRow2b.SetBackgroundColor(tcell.ColorRoyalBlue)
	footerRow2b.SetFieldBackgroundColor(tcell.ColorBlack)
	footerRow3 := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetTextColor(tcell.ColorLightYellow)
	footerRow3.SetBackgroundColor(tcell.ColorRoyalBlue).SetBorderPadding(0, 0, 1, 1)
	flexFooter2 := tview.NewFlex().SetDirection(tview.FlexColumn).AddItem(footerRow2a, 15, 1, false).AddItem(footerRow2b, 0, 1, false)
	flexFooter3 := tview.NewFlex().SetDirection(tview.FlexColumn).AddItem(footerRow3, 0, 1, false)
	flexBody := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(headerRow1, 1, 1, false).AddItem(headerRow2, 1, 1, false)

	// TODO Search box in footer

	rootDir := getRootDir()
	root := tview.NewTreeNode(rootDir)
	populate(root, rootDir)
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		path := rootDir
		if reference != nil {
			path, _ = reference.(string)
		}
		app.Stop()
		result <- path
	}).SetChangedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			return
		}
		path, ok := ref.(string)
		if ok {
			footerRow3.SetText(path)
		}
	}).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft {
			// TODO Go back
			tree.GetCurrentNode().Collapse()
			return nil
		}
		if event.Key() == tcell.KeyRight {
			currentNode := tree.GetCurrentNode()
			children := tree.GetCurrentNode().GetChildren()
			if len(children) == 0 {
				path := rootDir
				reference := currentNode.GetReference()
				if reference != nil {
					path = reference.(string)
				}
				populate(currentNode, path)
			} else {
				currentNode.SetExpanded(true)
			}
			return nil
		}
		if event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
			search := footerRow2b.GetText()
			if len(search) == 0 {
				return nil
			}
			footerRow2b.SetText(search[:len(search)-1])
			return nil
		}
		if event.Key() == tcell.KeyRune {
			footerRow2b.SetText(footerRow2b.GetText() + string(event.Rune()))
			return nil
		}
		if event.Key() == tcell.KeyESC || event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEscape {
			app.Stop()
			os.Exit(0)
			return nil
		}

		return event
	}).SetBackgroundColor(tcell.ColorRoyalBlue).SetBorderPadding(1, 1, 1, 1)

	// Goto current dir
	current, err := os.Getwd()
	if err != nil {
		footerRow3.SetText("error " + err.Error()) // TODO Error message area
	}
	navigateTo(tree, current)

	flexBody.AddItem(tree, 0, 1, false).AddItem(footerRow1, 1, 1, false).AddItem(flexFooter2, 1, 1, false).AddItem(flexFooter3, 1, 1, false)

	app.SetRoot(flexBody, true).EnableMouse(false).SetFocus(tree).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			app.Stop()
			os.Exit(0)
			return nil
		}
		return event
	})

	if err := app.Run(); err != nil {
		panic(err)
	}

	directory := <-result
	fmt.Println(directory)
}

func populate(target *tview.TreeNode, path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		// panic(err)
		// TODO Message
		return
	}
	for _, file := range files {
		node := tview.NewTreeNode(file.Name()).
			SetReference(filepath.Join(path, file.Name())).
			SetSelectable(file.IsDir())
		if file.IsDir() {
			target.AddChild(node)
		}
	}
}

func getRootDir() string {
	if runtime.GOOS == "windows" {
		wd, err := os.Getwd()
		if err != nil {
			return os.Getenv("SystemDrive") + string(os.PathSeparator)
		}
		return filepath.VolumeName(wd) + string(os.PathSeparator)
	}
	return "/"
}

func navigateTo(tree *tview.TreeView, path string) {
	pathParts := strings.Split(path, string(os.PathSeparator))
	children := tree.GetCurrentNode().GetChildren()
	for i := 0; i < len(pathParts); i++ {
		for j := 0; j < len(children); j++ {
			child := children[j]
			if child.GetText() == pathParts[i] {
				path := getRootDir()
				reference := child.GetReference()
				if reference != nil {
					path = reference.(string)
				}
				populate(child, path)
				child.Expand()
				tree.Move(j + 1)
				children = child.GetChildren()
			}
		}
	}
}
