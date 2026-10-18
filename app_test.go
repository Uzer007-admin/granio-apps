package main

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApp_FilterCompatible(t *testing.T) {
	l := AppList{
		"native":   {ID: "native", Requires: runtime.GOOS},
		"multiple": {ID: "multiple", Requires: runtime.GOOS + ",powerpc"},
		"other":    {ID: "other", Requires: "powerpc"},
		"portable": {ID: "portable"},
	}

	assert.Len(t, l, 4)
	assert.Equal(t, AppList{
		"native":   l["native"],
		"multiple": l["multiple"],
		"portable": l["portable"],
	}, l.filterCompatible())
}

func TestApp_IsCompatible(t *testing.T) {
	a := &App{Requires: "linux"}

	if runtime.GOOS == "linux" {
		assert.True(t, a.isCompatible())
	} else {
		assert.False(t, a.isCompatible())
	}

	a.Requires = runtime.GOOS
	assert.True(t, a.isCompatible())
}

func TestApp_IsCompatibleWithOS(t *testing.T) {
	a := &App{Requires: "darwin"}
	assert.True(t, a.isCompatibleWithOS("darwin"))
	assert.False(t, a.isCompatibleWithOS("powerpc"))

	a.Requires = "linux,powerpc"
	assert.True(t, a.isCompatibleWithOS("powerpc"))
	a.Requires = ""
	assert.True(t, a.isCompatibleWithOS("powerpc"))
}
