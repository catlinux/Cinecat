package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PhotoThumbnailer genera miniaturas de fotos con FFmpeg y las cachea por id y
// mtime, igual que las carátulas de música.
type PhotoThumbnailer struct {
	cacheDir string
	runner   ThumbnailRunner
	sem      chan struct{}
	wait     time.Duration
	timeout  time.Duration
	locks    sync.Map
}

func NewPhotoThumbnailer(cacheDir string) *PhotoThumbnailer {
	return &PhotoThumbnailer{
		cacheDir: strings.TrimSpace(cacheDir),
		runner:   runThumbnailFFmpeg,
		sem:      make(chan struct{}, thumbnailConcurrency),
		wait:     thumbnailWaitTimeout,
		timeout:  thumbnailGenerationTimeout,
	}
}

func (t *PhotoThumbnailer) WithRunner(runner ThumbnailRunner) *PhotoThumbnailer {
	if runner != nil {
		t.runner = runner
	}
	return t
}

func (t *PhotoThumbnailer) WithLimits(wait, timeout time.Duration) *PhotoThumbnailer {
	if wait > 0 {
		t.wait = wait
	}
	if timeout > 0 {
		t.timeout = timeout
	}
	return t
}

func (t *PhotoThumbnailer) Get(ctx context.Context, photoID int64, photoPath string, mtimes ...int64) (string, error) {
	if t == nil || t.cacheDir == "" {
		return "", ErrThumbnailUnavailable
	}
	mtime := int64(0)
	if len(mtimes) > 0 {
		mtime = mtimes[0]
	} else if stat, err := os.Stat(photoPath); err == nil {
		mtime = stat.ModTime().UnixNano()
	}
	target := filepath.Join(t.cacheDir, fmt.Sprintf("%d-%d.jpg", photoID, mtime))
	if usableThumbnail(target) {
		return target, nil
	}
	select {
	case t.sem <- struct{}{}:
	default:
		timer := time.NewTimer(t.wait)
		defer timer.Stop()
		select {
		case t.sem <- struct{}{}:
		case <-timer.C:
			return "", ErrThumbnailBusy
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	defer func() { <-t.sem }()

	keyLock, _ := t.locks.LoadOrStore(target, &sync.Mutex{})
	mutex := keyLock.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()
	if usableThumbnail(target) {
		return target, nil
	}
	if err := os.MkdirAll(t.cacheDir, 0o750); err != nil {
		return "", fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	file, err := os.CreateTemp(t.cacheDir, ".photo-*.jpg")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	temp := file.Name()
	_ = file.Close()
	_ = os.Remove(temp)

	genCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-i", photoPath,
		"-frames:v", "1",
		"-vf", "scale=400:400:force_original_aspect_ratio=decrease",
		"-q:v", "3",
		"-y", temp,
	}
	if err := t.runner(genCtx, args); err != nil {
		_ = os.Remove(temp)
		return "", fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	if !usableThumbnail(temp) {
		_ = os.Remove(temp)
		return "", ErrThumbnailUnavailable
	}
	if err := os.Rename(temp, target); err != nil {
		_ = os.Remove(temp)
		if usableThumbnail(target) {
			return target, nil
		}
		return "", fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	return target, nil
}

func (a *API) WithPhotoThumbnails(cacheDir string) *API {
	a.photoThumbnails = NewPhotoThumbnailer(cacheDir)
	return a
}

func (a *API) WithPhotoThumbnailRunner(runner ThumbnailRunner) *API {
	if a.photoThumbnails != nil {
		a.photoThumbnails.WithRunner(runner)
	}
	return a
}

func (a *API) WithPhotoThumbnailLimits(wait, timeout time.Duration) *API {
	if a.photoThumbnails != nil {
		a.photoThumbnails.WithLimits(wait, timeout)
	}
	return a
}
