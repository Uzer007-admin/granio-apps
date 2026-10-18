package main

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadAppListFromTestData() (io.ReadCloser, error) {
	res, err := os.Open(filepath.Join("testdata", "list.json"))
	if err != nil {
		return nil, err
	}

	return res, nil
}

func TestParseAppList(t *testing.T) {
	res, err := loadAppListFromTestData()
	require.NoError(t, err)
	defer res.Close()
	list, err := parseAppList(res)
	require.NoError(t, err)
	if runtime.GOOS != "linux" {
		assert.Len(t, list, 9)
		assert.NotContains(t, list, "xyz.andy.beebui")
		for _, app := range list {
			assert.True(t, app.isCompatible())
		}
		return
	}
	assert.Len(t, list, 10)

	app := list["xyz.andy.beebui"]
	assert.Equal(t, "xyz.andy.beebui", app.ID)
	assert.Equal(t, "BeebUI", app.Name)
	assert.Equal(t, "https://github.com/andydotxyz/beebui/blob/master/icon.png?raw=true", app.Icon)
	assert.True(t, time.Date(2019, 03, 17, 19, 32, 14, 0, time.UTC).Equal(app.Date))
	assert.Equal(t, "A BBC Micro Emulator based on Fyne and skx/gobasic", app.Summary)
	assert.Equal(t, "https://apps.github.com/Uzer007-admin/granio-apps/beebui.html", app.URL)
	assert.Equal(t, "https://github.com/andydotxyz/beebui", app.Website)

	assert.NotNil(t, app.Screenshots)
	require.Len(t, app.Screenshots, 1)
	shot := app.Screenshots[0]
	assert.Equal(t, "https://github.com/andydotxyz/beebui/blob/master/beebui.png?raw=true", shot.Image)
	assert.Equal(t, "desktop", shot.Type)

	assert.Equal(t, "https://github.com/andydotxyz/beebui.git", app.Source.Git)
	assert.Equal(t, "github.com/andydotxyz/beebui/cmd/beebui", app.Source.Package)
	assert.Equal(t, "", app.Version)
	assert.Equal(t, "linux", app.Requires)
}
