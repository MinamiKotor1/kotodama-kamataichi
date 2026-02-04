package download

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"kotodama-kamataichi/internal/tunehub"
)

type Progress struct {
	Kind  string
	Bytes int64
	Total int64
}

type Result struct {
	Dir        string
	AudioPath  string
	CoverPath  string
	MetaPath   string
	LyricsPath string
}

type Downloader struct {
	HTTP *http.Client
}

func NewDownloader() *Downloader {
	return &Downloader{HTTP: &http.Client{Timeout: 60 * time.Second}}
}

func (d *Downloader) DownloadSong(ctx context.Context, rootDir string, item tunehub.ParseItem, onProgress func(Progress)) (Result, error) {
	if strings.TrimSpace(rootDir) == "" {
		return Result{}, errors.New("root dir required")
	}
	if strings.TrimSpace(item.ID) == "" {
		return Result{}, errors.New("missing song id")
	}
	if strings.TrimSpace(item.URL) == "" {
		return Result{}, errors.New("missing song url")
	}

	songDirName := sanitizeName(fmt.Sprintf("%s - %s [%s]", item.Info.Artist, item.Info.Name, item.ID))
	songDir := filepath.Join(rootDir, songDirName)
	if err := os.MkdirAll(songDir, 0o755); err != nil {
		return Result{}, err
	}

	metaPath := filepath.Join(songDir, "meta.json")
	if err := writeJSON(metaPath, item); err != nil {
		return Result{}, err
	}

	lyricsPath := ""
	if strings.TrimSpace(item.Lyrics) != "" {
		lyricsPath = filepath.Join(songDir, "lyrics.lrc")
		if err := os.WriteFile(lyricsPath, []byte(item.Lyrics), 0o644); err != nil {
			return Result{}, err
		}
	}

	audioExt := audioExtFromQuality(item.ActualQuality, item.Quality)
	audioName := sanitizeName(fmt.Sprintf("%s - %s", item.Info.Artist, item.Info.Name)) + audioExt
	audioPath := filepath.Join(songDir, audioName)

	var coverPath string
	coverURL := strings.TrimSpace(item.Cover)
	if coverURL != "" {
		coverPath = filepath.Join(songDir, "cover"+coverExtFromURL(coverURL))
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return downloadFile(gctx, d.http(), item.URL, audioPath, item.FileSize, func(p Progress) {
			p.Kind = "audio"
			if onProgress != nil {
				onProgress(p)
			}
		})
	})
	if coverURL != "" {
		g.Go(func() error {
			return downloadFile(gctx, d.http(), coverURL, coverPath, 0, func(p Progress) {
				p.Kind = "cover"
				if onProgress != nil {
					onProgress(p)
				}
			})
		})
	}

	if err := g.Wait(); err != nil {
		return Result{}, err
	}
	return Result{Dir: songDir, AudioPath: audioPath, CoverPath: coverPath, MetaPath: metaPath, LyricsPath: lyricsPath}, nil
}

func (d *Downloader) http() *http.Client {
	if d.HTTP != nil {
		return d.HTTP
	}
	return http.DefaultClient
}

func writeJSON(p string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(p, b, 0o644)
}

func audioExtFromQuality(actual, requested string) string {
	q := strings.ToLower(strings.TrimSpace(actual))
	if q == "" {
		q = strings.ToLower(strings.TrimSpace(requested))
	}
	if strings.Contains(q, "flac") {
		return ".flac"
	}
	return ".mp3"
}

func coverExtFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err == nil {
		ext := strings.ToLower(path.Ext(u.Path))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp":
			return ext
		}
	}
	return ".jpg"
}

func downloadFile(ctx context.Context, client *http.Client, rawURL string, dst string, expectedTotal int64, progress func(Progress)) (err error) {
	if strings.TrimSpace(rawURL) == "" {
		return errors.New("missing url")
	}
	if strings.TrimSpace(dst) == "" {
		return errors.New("missing dst")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("http %d", res.StatusCode)
	}

	total := res.ContentLength
	if total <= 0 && expectedTotal > 0 {
		total = expectedTotal
	}
	part := dst + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		if err != nil {
			_ = os.Remove(part)
		}
	}()

	buf := make([]byte, 32*1024)
	var n int64
	for {
		rn, rerr := res.Body.Read(buf)
		if rn > 0 {
			wn, werr := f.Write(buf[:rn])
			if werr != nil {
				return werr
			}
			n += int64(wn)
			if progress != nil {
				progress(Progress{Bytes: n, Total: total})
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			return rerr
		}
	}
	if cerr := f.Close(); cerr != nil {
		return cerr
	}
	if err := os.Rename(part, dst); err != nil {
		return err
	}
	return nil
}
