package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"strings"
	"time"

	"github.com/Uzer007-admin/granio/v2"
	"github.com/Uzer007-admin/granio/v2/canvas"
	"github.com/Uzer007-admin/granio/v2/container"
	"github.com/Uzer007-admin/granio/v2/layout"
	"github.com/Uzer007-admin/granio/v2/storage"
	"github.com/Uzer007-admin/granio/v2/theme"
	"github.com/Uzer007-admin/granio/v2/widget"
)

type Feature struct {
	ID, Description, Image string

	Background, Color, Icon string
}

func makeFeatured(ctx context.Context, apps AppList, choose func(string)) *granio.Container {
	frame := container.NewStack()
	go func() {
		data, err := downloadData(ctx, "https://apps.fyne.io/api/v1/featured.json", 2<<20)
		if err != nil {
			if ctx.Err() == nil {
				granio.LogError("Failed to load featured apps", err)
			}
			return
		}
		list, err := parseFeatured(bytes.NewReader(data))
		if err != nil {
			granio.LogError("Failed to parse featured apps", err)
			return
		}
		visible := make([]Feature, 0, len(list))
		for _, item := range list {
			if _, ok := apps[item.ID]; ok {
				visible = append(visible, item)
			}
		}
		granio.Do(func() {
			if ctx.Err() == nil {
				frame.Add(buildFeatured(ctx, visible, apps, choose))
			}
		})
	}()
	return frame
}

func buildFeatured(ctx context.Context, list []Feature, apps AppList, choose func(string)) *granio.Container {
	if len(list) == 0 {
		return &granio.Container{}
	}

	// Build all slide widgets up front
	slides := make([]granio.CanvasObject, len(list))
	for i, item := range list {
		app := apps[item.ID]
		slides[i] = makeFeaturedSlide(ctx, app, item, choose)
	}

	// Show first slide
	current := 0
	slideContainer := container.NewStack(slides[0])

	// Dot indicators
	dots := make([]*canvas.Rectangle, len(list))
	dotObjects := make([]granio.CanvasObject, len(list))
	for i := range list {
		dot := canvas.NewRectangle(color.NRGBA{R: 180, G: 180, B: 180, A: 255})
		dot.SetMinSize(granio.NewSquareSize(8))
		dot.CornerRadius = 4
		dots[i] = dot
		dotObjects[i] = dot
	}
	dots[0].FillColor = color.NRGBA{R: 100, G: 149, B: 237, A: 255}

	showSlide := func(index int) {
		if index < 0 || index >= len(slides) {
			return
		}
		dots[current].FillColor = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
		dots[current].Refresh()
		current = index
		dots[current].FillColor = color.NRGBA{R: 100, G: 149, B: 237, A: 255}
		dots[current].Refresh()

		slideContainer.RemoveAll()
		slideContainer.Add(slides[current])
		slideContainer.Refresh()
	}

	// Auto-advance timer, resets on manual interaction
	autoTimer := time.NewTicker(5 * time.Second)
	go func() {
		defer autoTimer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-autoTimer.C:
				granio.Do(func() {
					if ctx.Err() != nil {
						return
					}
					n := current + 1
					if n >= len(slides) {
						n = 0
					}
					showSlide(n)
				})
			}
		}
	}()

	resetTimer := func() {
		autoTimer.Reset(5 * time.Second)
	}

	// Wrap dots in tappable buttons
	dotContainer := container.NewHBox(layout.NewSpacer())
	for i, dotObj := range dotObjects {
		idx := i
		tap := widget.NewButton("", func() {
			resetTimer()
			showSlide(idx)
		})
		tap.Importance = widget.LowImportance
		dotContainer.Add(container.NewStack(tap, container.NewCenter(dotObj)))
	}
	dotContainer.Add(layout.NewSpacer())

	prev := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		resetTimer()
		next := current - 1
		if next < 0 {
			next = len(slides) - 1
		}
		showSlide(next)
	})
	prev.Importance = widget.LowImportance

	next := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		resetTimer()
		n := current + 1
		if n >= len(slides) {
			n = 0
		}
		showSlide(n)
	})
	next.Importance = widget.LowImportance

	carousel := container.NewBorder(nil, nil, prev, next, slideContainer)

	return container.NewVBox(carousel, dotContainer)
}

func makeFeaturedSlide(ctx context.Context, app App, item Feature, choose func(string)) granio.CanvasObject {
	var obj granio.CanvasObject
	if item.Image != "" {
		path := item.Image
		if strings.HasPrefix(path, "/") {
			path = "https://apps.fyne.io" + path
		}

		u, _ := storage.ParseURI(path)
		img := canvas.NewImageFromURIAsync(ctx, u, nil)
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(granio.NewSize(0, 150))

		obj = img
	} else {
		bg, fg := &color.NRGBA{}, &color.NRGBA{}
		bg.A = 0xff
		_, _ = fmt.Sscanf(item.Background, "#%02x%02x%02x", &bg.R, &bg.G, &bg.B)
		fg.A = 0xff
		_, _ = fmt.Sscanf(item.Color, "#%02x%02x%02x", &fg.R, &fg.G, &fg.B)

		path := item.Icon
		if path == "" {
			path = app.Icon
		}
		if strings.HasPrefix(path, "/") {
			path = "https://apps.fyne.io" + path
		}

		u, _ := storage.ParseURI(path)
		icon := canvas.NewImageFromURIAsync(ctx, u, nil)
		icon.SetMinSize(granio.NewSquareSize(56))

		description := canvas.NewText(item.Description, fg)
		description.TextSize = 13
		name := canvas.NewText(app.Name, fg)
		name.TextSize = 18
		name.TextStyle.Bold = true

		bgRect := canvas.NewRectangle(bg)
		bgRect.SetMinSize(granio.NewSize(0, 150))
		bgRect.CornerRadius = 8

		content := container.NewVBox(
			container.NewCenter(container.NewHBox(icon, name)),
			container.NewCenter(description),
		)
		obj = container.NewStack(bgRect, container.NewCenter(content))
	}

	tap := widget.NewButton("", func() {
		choose(app.ID)
	})
	tap.Importance = widget.LowImportance

	return container.NewStack(tap, obj)
}

func parseFeatured(reader io.Reader) ([]Feature, error) {
	decode := json.NewDecoder(reader)

	var list []Feature
	err := decode.Decode(&list)
	if err != nil {
		return nil, err
	}

	return list, nil
}
