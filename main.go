//go:generate fyne bundle -o bundled.go Icon.png

package main

import (
	"context"

	"github.com/Uzer007-admin/granio/v2"
	"github.com/Uzer007-admin/granio/v2/app"
	"github.com/Uzer007-admin/granio/v2/container"
	"github.com/Uzer007-admin/granio/v2/widget"
)

func main() {
	a := app.NewWithID("io.granio.apps")
	a.SetIcon(resourceIconPng)
	w := a.NewWindow("Приложения Granio")

	ctx, cancel := context.WithCancel(context.Background())
	w.SetOnClosed(func() {
		cancel()
		cardLoaders.Range(func(key, _ any) bool { cardLoaders.Delete(key); return true })
	})
	var load func()
	load = func() {
		w.SetContent(container.NewCenter(widget.NewLabel("Loading applications…")))
		go func() {
			cacheFile, _ := catalogCachePath()
			apps, err := loadCatalog(ctx, catalogURL, cacheFile)
			granio.Do(func() {
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					w.SetContent(container.NewCenter(container.NewVBox(widget.NewLabel("Could not load applications"), widget.NewButton("Retry", load))))
					return
				}
				w.SetContent(loadUI(ctx, apps, w))
			})
		}()
	}
	load()
	w.Resize(granio.NewSize(680, 560))

	w.ShowAndRun()
}
