package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/atomic"
)

const catalogURL = "https://apps.fyne.io/api/v1/list.json"
const maxCatalogBytes = 8 << 20

var appsHTTPClient = &http.Client{Timeout: 20 * time.Second}

func catalogCachePath() (string, error) {
	directory, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "fyne", "apps", "list.json"), nil
}

func downloadData(ctx context.Context, location string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	response, err := appsHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds %d bytes", limit)
	}
	return data, nil
}

func loadCatalog(ctx context.Context, location, cacheFile string) (AppList, error) {
	data, webErr := downloadData(ctx, location, maxCatalogBytes)
	if webErr == nil {
		list, err := parseAppList(bytes.NewReader(data))
		if err == nil {
			// Cache failure does not prevent use of a valid network response.
			if cacheFile != "" && os.MkdirAll(filepath.Dir(cacheFile), 0700) == nil {
				_ = atomic.WriteFile(cacheFile, bytes.NewReader(data))
			}
			return list, nil
		}
		webErr = err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	file, cacheErr := os.Open(cacheFile)
	if cacheErr != nil {
		return nil, fmt.Errorf("catalog: %w; cache: %v", webErr, cacheErr)
	}
	defer file.Close()
	data, cacheErr = io.ReadAll(io.LimitReader(file, maxCatalogBytes+1))
	if cacheErr == nil && len(data) > maxCatalogBytes {
		cacheErr = fmt.Errorf("cached catalog exceeds byte limit")
	}
	if cacheErr != nil {
		return nil, cacheErr
	}
	return parseAppList(bytes.NewReader(data))
}
