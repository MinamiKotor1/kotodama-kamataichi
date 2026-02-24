package audiotag

import (
	"os"

	flac "github.com/go-flac/go-flac/v2"
	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
)

func tagFLAC(audioPath, coverPath string, meta Metadata) error {
	f, err := flac.ParseFile(audioPath)
	if err != nil {
		return err
	}

	var cmts *flacvorbis.MetaDataBlockVorbisComment
	cmtIdx := -1
	for idx, m := range f.Meta {
		if m.Type == flac.VorbisComment {
			cmts, err = flacvorbis.ParseFromMetaDataBlock(*m)
			if err != nil {
				return err
			}
			cmtIdx = idx
		}
	}
	if cmts == nil {
		cmts = flacvorbis.New()
	}

	cmts.Add(flacvorbis.FIELD_TITLE, meta.Title)
	cmts.Add(flacvorbis.FIELD_ARTIST, meta.Artist)
	cmts.Add(flacvorbis.FIELD_ALBUM, meta.Album)

	if meta.Lyrics != "" {
		cmts.Add("LYRICS", meta.Lyrics)
		cmts.Add("UNSYNCEDLYRICS", meta.Lyrics)
	}

	cmtBlock := cmts.Marshal()
	if cmtIdx >= 0 {
		f.Meta[cmtIdx] = &cmtBlock
	} else {
		f.Meta = append(f.Meta, &cmtBlock)
	}

	if coverPath != "" {
		mime := mimeFromExt(coverPath)
		if mime == "image/jpeg" || mime == "image/png" {
			imgData, err := os.ReadFile(coverPath)
			if err == nil {
				pic, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Cover", imgData, mime)
				if err == nil {
					picBlock := pic.Marshal()
					f.Meta = append(f.Meta, &picBlock)
				}
			}
		}
	}

	return f.Save(audioPath)
}
