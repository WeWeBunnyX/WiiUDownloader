package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	wiiudownloader "github.com/Xpl0itU/WiiUDownloader"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sqweek/dialog"
)

const (
	IN_QUEUE_COLUMN = 0
	NAME_COLUMN     = 1
	KIND_COLUMN     = 2
	TITLE_ID_COLUMN = 3
	REGION_COLUMN   = 4
)

type MainWindow struct {
	window                          *gtk.Window
	treeView                        *gtk.TreeView
	titles                          []wiiudownloader.TitleEntry
	searchEntry                     *gtk.Entry
	categoryButtons                 []*gtk.ToggleButton
	titleQueue                      []wiiudownloader.TitleEntry
	progressWindow                  wiiudownloader.ProgressWindow
	addToQueueButton                *gtk.Button
	decryptContents                 bool
	currentRegion                   uint8
	deleteEncryptedContentsCheckbox *gtk.CheckButton
}

func NewMainWindow(entries []wiiudownloader.TitleEntry) *MainWindow {
	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}

	win.SetTitle("WiiUDownloaderGo")
	win.SetDefaultSize(716, 400)
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	searchEntry, err := gtk.EntryNew()
	if err != nil {
		log.Fatal("Unable to create search entry:", err)
	}

	mainWindow := MainWindow{
		window:        win,
		titles:        entries,
		searchEntry:   searchEntry,
		currentRegion: wiiudownloader.MCP_REGION_EUROPE | wiiudownloader.MCP_REGION_JAPAN | wiiudownloader.MCP_REGION_USA,
	}

	searchEntry.Connect("changed", mainWindow.onSearchEntryChanged)

	return &mainWindow
}

func (mw *MainWindow) updateTitles(titles []wiiudownloader.TitleEntry) {
	store, err := gtk.ListStoreNew(glib.TYPE_BOOLEAN, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}

	for _, entry := range titles {
		if (mw.currentRegion & entry.Region) == 0 {
			continue
		}
		iter := store.Append()
		err = store.Set(iter,
			[]int{IN_QUEUE_COLUMN, NAME_COLUMN, KIND_COLUMN, TITLE_ID_COLUMN, REGION_COLUMN},
			[]interface{}{mw.isTitleInQueue(entry), entry.Name, wiiudownloader.GetFormattedKind(entry.TitleID), fmt.Sprintf("%016x", entry.TitleID), wiiudownloader.GetFormattedRegion(entry.Region)},
		)
		if err != nil {
			log.Fatal("Unable to set values:", err)
		}
	}
	mw.treeView.SetModel(store)
}

func (mw *MainWindow) ShowAll() {
	store, err := gtk.ListStoreNew(glib.TYPE_BOOLEAN, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatal("Unable to create list store:", err)
	}

	for _, entry := range mw.titles {
		if (mw.currentRegion & entry.Region) == 0 {
			continue
		}
		iter := store.Append()
		err = store.Set(iter,
			[]int{IN_QUEUE_COLUMN, NAME_COLUMN, KIND_COLUMN, TITLE_ID_COLUMN, REGION_COLUMN},
			[]interface{}{mw.isTitleInQueue(entry), entry.Name, wiiudownloader.GetFormattedKind(entry.TitleID), fmt.Sprintf("%016x", entry.TitleID), wiiudownloader.GetFormattedRegion(entry.Region)},
		)
		if err != nil {
			log.Fatal("Unable to set values:", err)
		}
	}

	mw.treeView, err = gtk.TreeViewNew()
	if err != nil {
		log.Fatal("Unable to create tree view:", err)
	}

	mw.treeView.SetModel(store)

	toggleRenderer, err := gtk.CellRendererToggleNew()
	if err != nil {
		log.Fatal("Unable to create cell renderer toggle:", err)
	}
	column, err := gtk.TreeViewColumnNewWithAttribute("Queue", toggleRenderer, "active", IN_QUEUE_COLUMN)
	if err != nil {
		log.Fatal("Unable to create tree view column:", err)
	}
	mw.treeView.AppendColumn(column)

	renderer, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal("Unable to create cell renderer:", err)
	}
	column, err = gtk.TreeViewColumnNewWithAttribute("Name", renderer, "text", NAME_COLUMN)
	if err != nil {
		log.Fatal("Unable to create tree view column:", err)
	}
	mw.treeView.AppendColumn(column)

	renderer, err = gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal("Unable to create cell renderer:", err)
	}
	column, err = gtk.TreeViewColumnNewWithAttribute("Kind", renderer, "text", KIND_COLUMN)
	if err != nil {
		log.Fatal("Unable to create tree view column:", err)
	}
	mw.treeView.AppendColumn(column)

	renderer, err = gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal("Unable to create cell renderer:", err)
	}
	column, err = gtk.TreeViewColumnNewWithAttribute("Title ID", renderer, "text", TITLE_ID_COLUMN)
	if err != nil {
		log.Fatal("Unable to create tree view column:", err)
	}
	mw.treeView.AppendColumn(column)

	column, err = gtk.TreeViewColumnNewWithAttribute("Region", renderer, "text", REGION_COLUMN)
	if err != nil {
		log.Fatal("Unable to create tree view column:", err)
	}
	mw.treeView.AppendColumn(column)

	mainvBox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	if err != nil {
		log.Fatal("Unable to create box:", err)
	}
	tophBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		log.Fatal("Unable to create box:", err)
	}

	mw.categoryButtons = make([]*gtk.ToggleButton, 0)
	for _, cat := range []string{"Game", "Update", "DLC", "Demo", "All"} {
		button, err := gtk.ToggleButtonNewWithLabel(cat)
		if err != nil {
			log.Fatal("Unable to create toggle button:", err)
			continue
		}
		tophBox.PackStart(button, false, false, 0)
		button.Connect("pressed", mw.onCategoryToggled)
		buttonLabel, _ := button.GetLabel()
		if buttonLabel == "Game" {
			button.SetActive(true)
		}
		mw.categoryButtons = append(mw.categoryButtons, button)
	}
	tophBox.PackEnd(mw.searchEntry, false, false, 0)

	mainvBox.PackStart(tophBox, false, false, 0)

	scrollable, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		log.Fatal("Unable to create scrolled window:", err)
	}
	scrollable.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	selection, _ := mw.treeView.GetSelection()
	selection.Connect("changed", mw.onSelectionChanged)
	scrollable.Add(mw.treeView)

	mainvBox.PackStart(scrollable, true, true, 0)

	bottomhBox, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		log.Fatal("Unable to create box:", err)
	}

	mw.addToQueueButton, err = gtk.ButtonNewWithLabel("Add to queue")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}

	downloadQueueButton, err := gtk.ButtonNewWithLabel("Download queue")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}

	decryptContentsCheckbox, err := gtk.CheckButtonNewWithLabel("Decrypt contents")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}

	mw.deleteEncryptedContentsCheckbox, err = gtk.CheckButtonNewWithLabel("Delete encrypted contents after decryption")
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}
	mw.deleteEncryptedContentsCheckbox.SetSensitive(false)

	mw.addToQueueButton.Connect("clicked", mw.onAddToQueueClicked)
	downloadQueueButton.Connect("clicked", func() {
		mw.progressWindow, err = wiiudownloader.CreateProgressWindow(mw.window)
		if err != nil {
			return
		}
		mw.progressWindow.Window.ShowAll()
		go mw.onDownloadQueueClicked()
	})
	decryptContentsCheckbox.Connect("clicked", mw.onDecryptContentsClicked)
	bottomhBox.PackStart(mw.addToQueueButton, false, false, 0)
	bottomhBox.PackStart(downloadQueueButton, false, false, 0)

	checkboxvBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	checkboxvBox.PackStart(decryptContentsCheckbox, false, false, 0)
	checkboxvBox.PackEnd(mw.deleteEncryptedContentsCheckbox, false, false, 0)

	bottomhBox.PackStart(checkboxvBox, false, false, 0)

	japanButton, err := gtk.CheckButtonNewWithLabel("Japan")
	japanButton.SetActive(true)
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}
	japanButton.Connect("clicked", func() {
		mw.onRegionChange(japanButton, wiiudownloader.MCP_REGION_JAPAN)
	})
	bottomhBox.PackEnd(japanButton, false, false, 0)

	usaButton, err := gtk.CheckButtonNewWithLabel("USA")
	usaButton.SetActive(true)
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}
	usaButton.Connect("clicked", func() {
		mw.onRegionChange(usaButton, wiiudownloader.MCP_REGION_USA)
	})
	bottomhBox.PackEnd(usaButton, false, false, 0)

	europeButton, err := gtk.CheckButtonNewWithLabel("Europe")
	europeButton.SetActive(true)
	if err != nil {
		log.Fatal("Unable to create button:", err)
	}
	europeButton.Connect("clicked", func() {
		mw.onRegionChange(europeButton, wiiudownloader.MCP_REGION_EUROPE)
	})
	bottomhBox.PackEnd(europeButton, false, false, 0)

	mainvBox.PackEnd(bottomhBox, false, false, 0)

	mw.window.Add(mainvBox)

	mw.window.ShowAll()
}

func (mw *MainWindow) onRegionChange(button *gtk.CheckButton, region uint8) {
	if button.GetActive() {
		mw.currentRegion = region | mw.currentRegion
	} else {
		mw.currentRegion = region ^ mw.currentRegion
	}
	mw.updateTitles(mw.titles)
}

func (mw *MainWindow) onSearchEntryChanged() {
	text, _ := mw.searchEntry.GetText()
	mw.filterTitles(text)
}

func (mw *MainWindow) filterTitles(filterText string) {
	store, err := mw.treeView.GetModel()
	if err != nil {
		log.Fatal("Unable to get tree view model:", err)
	}

	storeRef := store.(*gtk.ListStore)
	storeRef.Clear()

	for _, entry := range mw.titles {
		if strings.Contains(strings.ToLower(entry.Name), strings.ToLower(filterText)) ||
			strings.Contains(strings.ToLower(fmt.Sprintf("%016x", entry.TitleID)), strings.ToLower(filterText)) {
			iter := storeRef.Append()
			err := storeRef.Set(iter,
				[]int{NAME_COLUMN, KIND_COLUMN, TITLE_ID_COLUMN, REGION_COLUMN},
				[]interface{}{entry.Name, wiiudownloader.GetFormattedKind(entry.TitleID), fmt.Sprintf("%016x", entry.TitleID), wiiudownloader.GetFormattedRegion(entry.Region)},
			)
			if err != nil {
				log.Fatal("Unable to set values:", err)
			}
		}
	}
}

func (mw *MainWindow) onCategoryToggled(button *gtk.ToggleButton) {
	category, _ := button.GetLabel()
	mw.updateTitles(wiiudownloader.GetTitleEntries(wiiudownloader.GetCategoryFromFormattedCategory(category)))
	for _, catButton := range mw.categoryButtons {
		catButton.SetActive(false)
	}
	button.Activate()
}

func (mw *MainWindow) onSelectionChanged() {
	selection, err := mw.treeView.GetSelection()
	if err != nil {
		log.Fatal("Unable to get selection:", err)
	}
	model, iter, _ := selection.GetSelected()
	if iter != nil {
		tid, _ := model.ToTreeModel().GetValue(iter, TITLE_ID_COLUMN)
		name, _ := model.ToTreeModel().GetValue(iter, NAME_COLUMN)
		if tid != nil {
			if tidStr, err := tid.GetString(); err == nil {
				tidNum, _ := strconv.ParseUint(tidStr, 16, 64)
				nameStr, _ := name.GetString()
				isInQueue := mw.isTitleInQueue(wiiudownloader.TitleEntry{TitleID: tidNum, Name: nameStr})
				if isInQueue {
					mw.addToQueueButton.SetLabel("Remove from queue")
				} else {
					mw.addToQueueButton.SetLabel("Add to queue")
				}
			}
		}
	}
}

func (mw *MainWindow) onDecryptContentsClicked() {
	if mw.decryptContents {
		mw.decryptContents = false
		mw.deleteEncryptedContentsCheckbox.SetSensitive(false)
	} else {
		mw.decryptContents = true
		mw.deleteEncryptedContentsCheckbox.SetSensitive(true)
	}
}

func (mw *MainWindow) getDeleteEncryptedContents() bool {
	if mw.deleteEncryptedContentsCheckbox.GetSensitive() {
		if mw.deleteEncryptedContentsCheckbox.GetActive() {
			return true
		}
	}
	return false
}

func (mw *MainWindow) isTitleInQueue(title wiiudownloader.TitleEntry) bool {
	for _, entry := range mw.titleQueue {
		if entry.TitleID == title.TitleID {
			return true
		}
	}
	return false
}

func (mw *MainWindow) addToQueue(tid string, name string) {
	titleID, err := strconv.ParseUint(tid, 16, 64)
	if err != nil {
		log.Fatal("Unable to parse title ID:", err)
	}
	mw.titleQueue = append(mw.titleQueue, wiiudownloader.TitleEntry{TitleID: titleID, Name: name})
}

func (mw *MainWindow) removeFromQueue(tid string) {
	for i, entry := range mw.titleQueue {
		if fmt.Sprintf("%016x", entry.TitleID) == tid {
			mw.titleQueue = append(mw.titleQueue[:i], mw.titleQueue[i+1:]...)
			return
		}
	}
}

func (mw *MainWindow) onAddToQueueClicked() {
	selection, err := mw.treeView.GetSelection()
	if err != nil {
		log.Fatal("Unable to get selection:", err)
	}
	model, iter, _ := selection.GetSelected()
	if iter != nil {
		tid, _ := model.ToTreeModel().GetValue(iter, TITLE_ID_COLUMN)
		name, _ := model.ToTreeModel().GetValue(iter, NAME_COLUMN)
		if tid != nil {
			if tidStr, err := tid.GetString(); err == nil {
				nameStr, _ := name.GetString()
				tidNum, _ := strconv.ParseUint(tidStr, 16, 64)
				titleInQueue := mw.isTitleInQueue(wiiudownloader.TitleEntry{TitleID: tidNum, Name: nameStr})
				if titleInQueue {
					mw.removeFromQueue(tidStr)
					mw.addToQueueButton.SetLabel("Add to queue")
				} else {
					mw.addToQueue(tidStr, nameStr)
					mw.addToQueueButton.SetLabel("Remove from queue")
				}
				store, _ := mw.treeView.GetModel()
				path, _ := store.(*gtk.ListStore).GetPath(iter)
				queueModel, _ := mw.treeView.GetModel()
				queueModel.(*gtk.ListStore).SetValue(iter, IN_QUEUE_COLUMN, !titleInQueue)
				mw.treeView.SetCursor(path, mw.treeView.GetColumn(IN_QUEUE_COLUMN), false)
			}
		}
	}
}

func (mw *MainWindow) updateTitlesInQueue() {
	store, err := mw.treeView.GetModel()
	if err != nil {
		log.Fatal("Unable to get tree view model:", err)
	}

	storeRef := store.(*gtk.ListStore)

	iter, _ := storeRef.GetIterFirst()
	for iter != nil {
		tid, _ := storeRef.GetValue(iter, TITLE_ID_COLUMN)
		if tid != nil {
			if tidStr, err := tid.GetString(); err == nil {
				tidNum, _ := strconv.ParseUint(tidStr, 16, 64)
				isInQueue := mw.isTitleInQueue(wiiudownloader.TitleEntry{TitleID: tidNum})
				storeRef.SetValue(iter, IN_QUEUE_COLUMN, isInQueue)
			}
		}
		if !storeRef.IterNext(iter) {
			break
		}
	}
}

func (mw *MainWindow) onDownloadQueueClicked() {
	queueCancelled := false
	var wg sync.WaitGroup

	selectedPath, err := dialog.Directory().Title("Select a path to save the games to").Browse()
	if err != nil {
		return
	}

	for _, title := range mw.titleQueue {
		wg.Add(1)

		go func(title wiiudownloader.TitleEntry, selectedPath string, progressWindow *wiiudownloader.ProgressWindow) {
			defer wg.Done()

			tidStr := fmt.Sprintf("%016x", title.TitleID)
			titlePath := fmt.Sprintf("%s/%s [%s]", selectedPath, title.Name, tidStr)
			if err := wiiudownloader.DownloadTitle(tidStr, titlePath, mw.decryptContents, progressWindow, mw.getDeleteEncryptedContents()); err != nil {
				queueCancelled = true
			}
			mw.removeFromQueue(tidStr)
		}(title, selectedPath, &mw.progressWindow)

		if queueCancelled {
			break
		}

		wg.Wait()
	}
	mw.titleQueue = []wiiudownloader.TitleEntry{} // Clear the queue
	mw.progressWindow.Window.Close()
	mw.updateTitlesInQueue()
}

func Main() {
	gtk.Main()
}
