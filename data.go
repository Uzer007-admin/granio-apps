package main

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/Uzer007-admin/granio/v2"
)

const keyInstallPrefix = "installed."

type App struct {
	ID, Name, Icon         string
	Developer, Summary     string
	URL, Website, Category string
	Screenshots            []AppScreenshot

	Date    time.Time
	Version string

	Source   AppSource
	Requires string
}

type AppScreenshot struct {
	Image, Type string
}

type AppSource struct {
	Git, Package string
}

type AppList map[string]App

func installedVersion(a App) string {
	return granio.CurrentApp().Preferences().String(keyInstallPrefix + a.ID)
}

func markInstalled(a App) {
	ver := a.Version
	if ver == "" {
		ver = "latest"
	}
	granio.CurrentApp().Preferences().SetString(keyInstallPrefix+a.ID, ver)
}

func parseAppList(reader io.Reader) (AppList, error) {
	decode := json.NewDecoder(reader)

	var list []App
	err := decode.Decode(&list)
	if err != nil {
		return nil, err
	}

	appList := AppList{}
	for _, a := range list {
		appList[a.ID] = a
	}

	return appList.filterCompatible(), nil
}

func loadAppListFromCache() (io.ReadCloser, error) {
	path, err := catalogCachePath()
	if err != nil {
		return nil, err
	}
	res, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return res, nil
}
