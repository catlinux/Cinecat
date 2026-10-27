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

var (
	ErrCoverBusy        = fmt.Errorf("%w: demasiadas carátulas generándose", ErrThumbnailBusy)
	ErrCoverUnavailable = fmt.Errorf("%w: no se pudo generar la carátula", ErrThumbnailUnavailable)
)

// MusicCoverer extrae y cachea la carátula incrustada de una pista.
type MusicCoverer struct {
	cacheDir string
	runner   ThumbnailRunner
	sem      chan struct{}
	wait     time.Duration
	timeout  time.Duration
	locks    sync.Map
}

func NewMusicCoverer(cacheDir string) *MusicCoverer {
	return &MusicCoverer{
		cacheDir: strings.TrimSpace(cacheDir),
		runner:   runThumbnailFFmpeg,
		sem:      make(chan struct{}, 2),
		wait:     thumbnailWaitTimeout,
		timeout:  thumbnailGenerationTimeout,
	}
}

func (c *MusicCoverer) WithRunner(runner ThumbnailRunner) *MusicCoverer {
	if runner != nil {
		c.runner = runner
	}
	return c
}

func (c *MusicCoverer) WithLimits(wait, timeout time.Duration) *MusicCoverer {
	if wait > 0 {
		c.wait = wait
	}
	if timeout > 0 {
		c.timeout = timeout
	}
	return c
}

func (c *MusicCoverer) Get(ctx context.Context, albumID int64, trackPath string, mtimes ...int64) (string, error) {
	if c == nil || c.cacheDir == "" {
		return "", ErrCoverUnavailable
	}
	mtime := int64(0)
	if len(mtimes) > 0 {
		mtime = mtimes[0]
	} else if stat, err := os.Stat(trackPath); err == nil {
		mtime = stat.ModTime().UnixNano()
	}
	target := filepath.Join(c.cacheDir, fmt.Sprintf("%d-%d.jpg", albumID, mtime))
	if usableThumbnail(target) {
		return target, nil
	}
	select {
	case c.sem <- struct{}{}:
	default:
		timer := time.NewTimer(c.wait)
		defer timer.Stop()
		select {
		case c.sem <- struct{}{}:
		case <-timer.C:
			return "", ErrCoverBusy
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	defer func() { <-c.sem }()

	keyLock, _ := c.locks.LoadOrStore(target, &sync.Mutex{})
	mutex := keyLock.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()
	if usableThumbnail(target) {
		return target, nil
	}
	if err := os.MkdirAll(c.cacheDir, 0o750); err != nil {
		return "", fmt.Errorf("%w: %v", ErrCoverUnavailable, err)
	}
	file, err := os.CreateTemp(c.cacheDir, ".cover-*.jpg")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCoverUnavailable, err)
	}
	temp := file.Name()
	_ = file.Close()
	_ = os.Remove(temp)

	genCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-i", trackPath,
		"-map", "0:v:0",
		"-frames:v", "1",
		"-q:v", "2",
		"-y", temp,
	}
	if err := c.runner(genCtx, args); err != nil {
		_ = os.Remove(temp)
		return "", fmt.Errorf("%w: %v", ErrCoverUnavailable, err)
	}
	if !usableThumbnail(temp) {
		_ = os.Remove(temp)
		return "", ErrCoverUnavailable
	}
	if err := os.Rename(temp, target); err != nil {
		_ = os.Remove(temp)
		if usableThumbnail(target) {
			return target, nil
		}
		return "", fmt.Errorf("%w: %v", ErrCoverUnavailable, err)
	}
	return target, nil
}

func imageMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func (a *API) WithMusicCovers(cacheDir string) *API {
	a.musicCovers = NewMusicCoverer(cacheDir)
	return a
}

func (a *API) WithMusicCoverRunner(runner ThumbnailRunner) *API {
	if a.musicCovers != nil {
		a.musicCovers.WithRunner(runner)
	}
	return a
}

func (a *API) WithMusicCoverLimits(wait, timeout time.Duration) *API {
	if a.musicCovers != nil {
		a.musicCovers.WithLimits(wait, timeout)
	}
	return a
}
