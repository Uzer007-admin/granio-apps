package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCatalogPersistsValidDataAndSurvivesNetworkFailure(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache", "list.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/valid":
			_, _ = w.Write([]byte(`[{"ID":"example.app","Name":"Example"}]`))
		case "/broken":
			_, _ = w.Write([]byte(`{"error":"not a catalog"}`))
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	list, err := loadCatalog(context.Background(), server.URL+"/valid", cache)
	require.NoError(t, err)
	require.Contains(t, list, "example.app")
	before, err := os.ReadFile(cache)
	require.NoError(t, err)
	for _, path := range []string{"/broken", "/failed"} {
		list, err = loadCatalog(context.Background(), server.URL+path, cache)
		require.NoError(t, err)
		require.Contains(t, list, "example.app")
	}
	after, _ := os.ReadFile(cache)
	require.Equal(t, before, after, "failed response replaced a valid cache")
}

func TestAppDownloadBoundsAndCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			close(started)
			<-r.Context().Done()
			return
		}
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer server.Close()
	_, err := downloadData(context.Background(), server.URL, 99)
	require.Error(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := downloadData(ctx, server.URL+"/slow", 100); result <- err }()
	<-started
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("catalog request ignored cancellation")
	}
}
