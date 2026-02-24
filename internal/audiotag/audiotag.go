package audiotag

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Metadata struct {
	Title  string
	Artist string
	Album  string
	Lyrics string
}

func TagAudio(audioPath, coverPath string, meta Metadata) error {
	switch strings.ToLower(filepath.Ext(audioPath)) {
	case ".mp3":
		return tagMP3(audioPath, coverPath, meta)
	case ".flac":
		return tagFLAC(audioPath, coverPath, meta)
	default:
		return fmt.Errorf("unsupported audio format: %s", filepath.Ext(audioPath))
	}
}

func mimeFromExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	default:
		return "image/jpeg"
	}
}
