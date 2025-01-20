// TODO Add theming
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Arguments struct {
	Full bool // Expensive operation
}

func main() {
	arguments := parseArguments()

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

	rootDir := getRootDir()
	root := tview.NewTreeNode(rootDir)
	root.SetTextStyle(root.GetTextStyle().Background(tcell.ColorRoyalBlue))
	populate(root, rootDir, arguments)
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
			if tree.GetCurrentNode().IsExpanded() && len(tree.GetCurrentNode().GetChildren()) != 0 {
				// On key Left, collapse current directory.
				tree.GetCurrentNode().Collapse()
			} else {
				// On key Left, if current directory is collapsed or empty, go to parent directory.
				family := tree.GetPath(tree.GetCurrentNode())
				parentPosition := len(family) - 2
				if parentPosition > -1 {
					family[parentPosition].Collapse()
					tree.SetCurrentNode(family[parentPosition])
				}
			}
			return nil
		}
		if event.Key() == tcell.KeyRight {
			// On key Right, expand current directory
			currentNode := tree.GetCurrentNode()
			children := tree.GetCurrentNode().GetChildren()
			if len(children) == 0 {
				path := rootDir
				reference := currentNode.GetReference()
				if reference != nil {
					path = reference.(string)
				}
				populate(currentNode, path, arguments)
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
			// Search for node and set current
			footerRow2b.SetText(search[:len(search)-1])
			if child := findNodeWithPrefix(root, footerRow2b.GetText()); child != nil {
				tree.SetCurrentNode(child)
			}
			return nil
		}
		if event.Key() == tcell.KeyRune {
			// Search for node and set current
			footerRow2b.SetText(footerRow2b.GetText() + string(event.Rune()))
			if child := findNodeWithPrefix(root, footerRow2b.GetText()); child != nil {
				tree.SetCurrentNode(child)
			}
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
	navigateTo(tree, current, arguments)

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

func parseArguments() Arguments {
	result := Arguments{}

	osArgs := map[string]string{}
	for i := 1; i < len(os.Args); i = i + 2 {
		osArgs[os.Args[i]] = os.Args[i+1]
	}
	result.Full, _ = strconv.ParseBool(osArgs["--full"])

	return result
}

func findNodeWithPrefix(parent *tview.TreeNode, prefix string) *tview.TreeNode {
	children := parent.GetChildren()
	for _, child := range children {
		if strings.HasPrefix(child.GetText(), prefix) {
			return child
		}
		if child.IsExpanded() {
			if node := findNodeWithPrefix(child, prefix); node != nil {
				return node
			}
		}
	}

	return nil
}

func populate(target *tview.TreeNode, path string, arguments Arguments) {
	files, err := os.ReadDir(path)
	if err != nil {
		// panic(err)
		// TODO Message
		return
	}
	for _, file := range files {
		if file.IsDir() {
			node := tview.NewTreeNode(file.Name()).SetReference(filepath.Join(path, file.Name()))
			node.SetTextStyle(node.GetTextStyle().Background(tcell.ColorRoyalBlue)).SetIndent(4)
			target.AddChild(node)
			if arguments.Full {
				populate(node, path+string(os.PathSeparator)+file.Name(), arguments)
			}
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

func navigateTo(tree *tview.TreeView, path string, arguments Arguments) {
	pathParts := strings.Split(path, string(os.PathSeparator))
	children := tree.GetCurrentNode().GetChildren()
	var currentNode *tview.TreeNode
	for i := 0; i < len(pathParts); i++ {
		for j := 0; j < len(children); j++ {
			child := children[j]
			if child.GetText() == pathParts[i] {
				path := getRootDir()
				reference := child.GetReference()
				if reference != nil {
					path = reference.(string)
				}
				if !arguments.Full {
					populate(child, path, arguments)
				}
				child.Expand()
				currentNode = child
				children = child.GetChildren()
			}
		}
	}
	tree.SetCurrentNode(currentNode)
}

func StringArrayContains(data []string, input string, caseSensitive bool) bool {
	for _, value := range data {
		if caseSensitive && value == input {
			return true
		}
		if !caseSensitive && strings.EqualFold(value, input) {
			return true
		}
	}
	return false
}
