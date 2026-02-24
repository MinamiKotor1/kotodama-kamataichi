package audiotag

import (
	"os"

	"github.com/bogem/id3v2/v2"
)

func tagMP3(audioPath, coverPath string, meta Metadata) error {
	tag, err := id3v2.Open(audioPath, id3v2.Options{Parse: false})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.SetDefaultEncoding(id3v2.EncodingUTF8)
	tag.SetTitle(meta.Title)
	tag.SetArtist(meta.Artist)
	tag.SetAlbum(meta.Album)

	if meta.Lyrics != "" {
		tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "und",
			ContentDescriptor: "",
			Lyrics:            meta.Lyrics,
		})
	}

	if coverPath != "" {
		pic, err := os.ReadFile(coverPath)
		if err == nil {
			tag.AddAttachedPicture(id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    mimeFromExt(coverPath),
				PictureType: id3v2.PTFrontCover,
				Description: "Cover",
				Picture:     pic,
			})
		}
	}

	return tag.Save()
}
