package main

import (
	"context"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/Uzer007-admin/granio/v2"
	"github.com/Uzer007-admin/granio/v2/canvas"
	"github.com/Uzer007-admin/granio/v2/cmd/granio/commands"
	"github.com/Uzer007-admin/granio/v2/container"
	"github.com/Uzer007-admin/granio/v2/dialog"
	"github.com/Uzer007-admin/granio/v2/layout"
	"github.com/Uzer007-admin/granio/v2/storage"
	"github.com/Uzer007-admin/granio/v2/theme"
	"github.com/Uzer007-admin/granio/v2/widget"
)

type welcome struct {
	shownApp            App
	name, summary, date *widget.Label
	developer, version  *widget.Label
	link                *widget.Hyperlink
	icon                *canvas.Image

	screenshots      [5]*canvas.Image
	screenScroll     *container.Scroll
	install          *widget.Button
	iconLoader       imageLoader
	screenshotLoader [5]imageLoader
}

func (w *welcome) loadAppDetail(app App) {
	w.shownApp = app

	w.name.SetText(app.Name)
	w.developer.SetText(app.Developer)
	w.version.SetText(app.Version)
	w.date.SetText(app.Date.Format("02 Jan 2006"))
	w.summary.SetText(app.Summary)

	w.icon.Resource = nil
	w.icon.Image = nil
	w.icon.Refresh()
	for _, scr := range w.screenshots {
		scr.Resource = nil
		scr.Image = nil
		scr.Refresh()
		scr.Hide()
	}

	w.iconLoader.load(w.icon, app.Icon)

	for i := 0; i < len(w.screenshots); i++ {
		w.screenshots[i].Resource = nil
		w.screenshots[i].Image = nil
		w.screenshots[i].Refresh()

		if i < len(app.Screenshots) {
			w.screenshots[i].Show()
			w.screenshotLoader[i].load(w.screenshots[i], app.Screenshots[i].Image)
		} else {
			w.screenshotLoader[i].load(w.screenshots[i], "") // cancel any in-flight
			w.screenshots[i].Hide()
		}
	}
	w.screenScroll.ScrollToTop()

	parsed, err := url.Parse(app.Website)
	if err != nil {
		w.link.SetText("")
		return
	}
	w.link.SetText(parsed.Host)
	w.link.SetURL(parsed)

	installedVer := installedVersion(app)
	installed := installedVer != "" && installedVer == app.Version
	if installed || app.Source.Package == "github.com/Uzer007-admin/granio-apps" {
		w.install.SetText("Installed")
		w.install.Disable()
	} else if installedVer == "" || app.Version == "" {
		w.install.SetText("Install")
		w.install.Enable()
	} else {
		w.install.SetText("Upgrade")
		w.install.Enable()
	}
}

// imageLoader manages cancellable image loading for a single canvas.Image.
// Calling load cancels any in-flight request for this image before starting a new one.
type imageLoader struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	parent context.Context
}

func (l *imageLoader) load(img *canvas.Image, location string) {
	l.mu.Lock()
	if l.cancel != nil {
		l.cancel()
	}
	img.Resource = nil
	img.Image = nil
	img.Refresh()
	if location == "" {
		l.cancel = nil
		l.mu.Unlock()
		return
	}
	parent := l.parent
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	l.cancel = cancel
	l.mu.Unlock()

	go func() {
		uri, err := storage.ParseURI(location)
		var loaded *canvas.Image
		if err == nil {
			loaded, err = canvas.LoadImage(ctx, uri)
		}
		if ctx.Err() != nil {
			return // cancelled, don't touch the image
		}
		granio.Do(func() {
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				img.Resource = theme.WarningIcon()
			} else {
				img.Resource, img.Image = loaded.Resource, loaded.Image
			}
			img.Refresh()
		})
	}()
}

// appCardLayout positions an icon centered above a label
type appCardLayout struct {
	icon, label granio.CanvasObject
}

func (a *appCardLayout) Layout(_ []granio.CanvasObject, size granio.Size) {
	iconSize := float32(48)
	labelH := a.label.MinSize().Height
	totalH := iconSize + 4 + labelH
	topPad := (size.Height - totalH) / 2
	if topPad < 4 {
		topPad = 4
	}

	a.icon.Resize(granio.NewSize(iconSize, iconSize))
	a.icon.Move(granio.NewPos((size.Width-iconSize)/2, topPad))

	a.label.Resize(granio.NewSize(size.Width, labelH))
	a.label.Move(granio.NewPos(0, topPad+iconSize+4))
}

func (a *appCardLayout) MinSize(_ []granio.CanvasObject) granio.Size {
	return granio.NewSize(110, 100)
}

func makeAppCardTemplate() granio.CanvasObject {
	icon := &canvas.Image{}
	icon.SetMinSize(granio.NewSquareSize(48))
	icon.FillMode = canvas.ImageFillContain

	name := widget.NewLabel("Template")
	name.Alignment = granio.TextAlignCenter
	name.Truncation = granio.TextTruncateEllipsis

	cardContent := container.New(&appCardLayout{icon: icon, label: name}, icon, name)
	return container.NewPadded(cardContent)
}

// cardLoaders tracks an imageLoader per grid cell icon so recycled cells cancel prior loads.
var cardLoaders sync.Map

func updateAppCard(ctx context.Context, obj granio.CanvasObject, app App) {
	padded := obj.(*granio.Container)
	card := padded.Objects[0].(*granio.Container) // the appCardLayout container
	icon := card.Objects[0].(*canvas.Image)
	label := card.Objects[1].(*widget.Label)

	label.SetText(app.Name)
	icon.Resource = nil
	icon.Image = nil
	icon.Refresh()

	val, _ := cardLoaders.LoadOrStore(icon, &imageLoader{parent: ctx})
	loader := val.(*imageLoader)
	loader.load(icon, app.Icon)
}

// detailHeaderLayout places the icon to the left with info to its right
type detailHeaderLayout struct {
	icon, info granio.CanvasObject
}

func (d *detailHeaderLayout) Layout(_ []granio.CanvasObject, size granio.Size) {
	iconSize := float32(80)
	d.icon.Resize(granio.NewSize(iconSize, iconSize))
	d.icon.Move(granio.NewPos(0, 0))

	infoW := size.Width - iconSize - 12
	if infoW < 0 {
		infoW = 0
	}
	d.info.Resize(granio.NewSize(infoW, size.Height))
	d.info.Move(granio.NewPos(iconSize+12, 0))
}

func (d *detailHeaderLayout) MinSize(_ []granio.CanvasObject) granio.Size {
	infoMin := d.info.MinSize()
	h := infoMin.Height
	if h < 80 {
		h = 80
	}
	return granio.NewSize(80+12+infoMin.Width, h)
}

func loadUI(ctx context.Context, apps AppList, win granio.Window) granio.CanvasObject {
	w := &welcome{}
	w.iconLoader.parent = ctx
	for i := range w.screenshotLoader {
		w.screenshotLoader[i].parent = ctx
	}
	w.name = widget.NewLabel("")
	w.name.TextStyle.Bold = true
	w.developer = widget.NewLabel("")
	w.link = widget.NewHyperlink("", nil)
	w.summary = widget.NewLabel("")
	w.summary.Wrapping = granio.TextWrapWord
	w.version = widget.NewLabel("")
	w.date = widget.NewLabel("")
	w.icon = &canvas.Image{}
	w.icon.SetMinSize(granio.NewSquareSize(80))
	w.icon.FillMode = canvas.ImageFillContain
	makeScreenshots(w)

	// Build the detail view
	w.install = widget.NewButton("Install", nil) // callback set below
	w.install.Importance = widget.HighImportance

	infoBlock := container.NewVBox(
		w.name,
		w.developer,
		container.NewHBox(w.link, layout.NewSpacer(), w.install),
	)
	w.name.TextStyle.Bold = true

	header := container.New(&detailHeaderLayout{icon: w.icon, info: infoBlock}, w.icon, infoBlock)

	metaLine := container.NewHBox(
		widget.NewLabel("Version:"),
		w.version,
		layout.NewSpacer(),
		w.date,
	)

	w.screenScroll = container.NewHScroll(container.NewHBox(
		w.screenshots[0], w.screenshots[1], w.screenshots[2], w.screenshots[3], w.screenshots[4]))

	detailContent := container.NewVBox(
		header,
		widget.NewSeparator(),
		w.summary,
		widget.NewSeparator(),
		metaLine,
		w.screenScroll,
	)
	detailScroll := container.NewVScroll(detailContent)

	// Build the category tabs and app grid
	categories := sortedCategories(apps)
	currentSort := sortByDate
	currentCategory := ""
	filteredIDs := sortedApps(apps, currentCategory, currentSort)

	var detailView, browseView *granio.Container

	appGrid := widget.NewGridWrap(
		func() int {
			return len(filteredIDs)
		},
		makeAppCardTemplate,
		func(id widget.GridWrapItemID, obj granio.CanvasObject) {
			if int(id) < len(filteredIDs) {
				updateAppCard(ctx, obj, apps[filteredIDs[id]])
			}
		},
	)
	appGrid.OnSelected = func(id widget.GridWrapItemID) {
		if int(id) < len(filteredIDs) {
			appID := filteredIDs[id]
			w.loadAppDetail(apps[appID])
			detailView.Show()
			browseView.Hide()
		}
		appGrid.UnselectAll()
	}

	filterGrid := func(category string) {
		currentCategory = category
		filteredIDs = sortedApps(apps, currentCategory, currentSort)
		appGrid.Refresh()
		appGrid.ScrollToTop()
	}

	catTabs := container.NewAppTabs(
		container.NewTabItem("All", layout.NewSpacer()),
	)
	for _, cat := range categories {
		title := strings.ToUpper(cat[:1]) + cat[1:]
		catTabs.Append(container.NewTabItem(title, layout.NewSpacer()))
	}
	catTabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "All" {
			filterGrid("")
		} else {
			filterGrid(strings.ToLower(tab.Text))
		}
	}

	// Featured banner
	featuredBanner := makeFeatured(ctx, apps, func(id string) {
		w.loadAppDetail(apps[id])
		detailView.Show()
		browseView.Hide()
	})

	// Section headers
	featuredHeader := makeSectionHeader("Featured")
	browseHeader := makeSectionHeader("Browse Apps")

	sortSelect := widget.NewSelect([]string{sortByDate, sortByName}, func(val string) {
		currentSort = val
		filteredIDs = sortedApps(apps, currentCategory, currentSort)
		appGrid.Refresh()
		appGrid.ScrollToTop()
	})
	sortSelect.Selected = currentSort

	browseRow := container.NewHBox(browseHeader, layout.NewSpacer(), sortSelect)

	topSection := container.NewVBox(
		featuredHeader,
		featuredBanner,
		widget.NewSeparator(),
		browseRow,
		catTabs,
	)

	browseView = container.NewBorder(topSection, nil, nil, nil, appGrid)

	backBtn := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
		browseView.Show()
		detailView.Hide()
	})
	backBtn.Importance = widget.LowImportance

	detailView = container.NewBorder(
		container.NewHBox(backBtn),
		nil, nil, nil,
		detailScroll,
	)

	// Wire up install button
	w.install.OnTapped = func() {
		target := w.shownApp
		bar := widget.NewProgressBarInfinite()
		content := container.NewVBox(widget.NewLabel("Please wait while the app is installed"), bar)
		prog := dialog.NewCustomWithoutButtons("Downloading...", content, win)
		bar.Start()
		prog.Show()

		go func() {
			get := commands.NewGetter()
			tmpIcon := downloadIcon(target.Icon)
			if tmpIcon != "" {
				defer os.Remove(tmpIcon)
			}
			get.SetIcon(tmpIcon)
			get.SetAppID(target.ID)
			err := get.Get(target.Source.Package)

			granio.Do(func() {
				prog.Hide()
				bar.Stop()
				if err != nil {
					dialog.ShowError(err, win)
				} else {
					dialog.ShowInformation("Installed", "App was installed successfully :)", win)
					markInstalled(target)
					if w.shownApp.ID == target.ID {
						w.loadAppDetail(target)
					}
				}
				_ = os.Remove(tmpIcon)
			})
		}()
	}

	detailView.Hide()
	return container.NewStack(browseView, detailView)
}

func makeSectionHeader(title string) granio.CanvasObject {
	label := canvas.NewText(title, color.NRGBA{R: 100, G: 149, B: 237, A: 255})
	label.TextSize = 18
	label.TextStyle.Bold = true
	return container.NewPadded(label)
}

func makeScreenshots(w *welcome) {
	for i := 0; i < len(w.screenshots); i++ {
		img := &canvas.Image{}
		img.SetMinSize(granio.NewSize(320, 240))
		img.FillMode = canvas.ImageFillContain

		w.screenshots[i] = img
	}
}

func sortedCategories(apps AppList) []string {
	catMap := map[string]bool{}
	for _, a := range apps {
		if a.Category != "" {
			catMap[a.Category] = true
		}
	}
	var cats []string
	for c := range catMap {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

const (
	sortByDate = "Newest"
	sortByName = "A-Z"
)

func sortedApps(apps AppList, category, sortMode string) []string {
	var ids []string
	for id, a := range apps {
		if category == "" || a.Category == category {
			ids = append(ids, id)
		}
	}
	switch sortMode {
	case sortByName:
		sort.Slice(ids, func(i, j int) bool {
			return strings.ToLower(apps[ids[i]].Name) < strings.ToLower(apps[ids[j]].Name)
		})
	default: // sortByDate – newest first
		sort.Slice(ids, func(i, j int) bool {
			return apps[ids[i]].Date.After(apps[ids[j]].Date)
		})
	}
	return ids
}

func downloadIcon(url string) string {
	data, err := downloadData(context.Background(), url, 16<<20)
	if err != nil {
		granio.LogError("Failed to access icon url: "+url, err)
		return ""
	}
	file, err := os.CreateTemp("", "Fyne-Icon-*.png")
	if err != nil {
		granio.LogError("Failed to create icon file", err)
		return ""
	}

	tmp := file.Name()
	_, err = file.Write(data)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		granio.LogError("Failed to get write icon to: "+tmp, err)
		return ""
	}

	return tmp
}
