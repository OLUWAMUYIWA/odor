package formats

import (
	"crypto/sha1"
	"fmt"
	"time"
)

// https://wiki.theory.org/index.php/BitTorrentSpecification

type MetaInfo struct {
	Info InfoDict 
	Announce string // url of the tracker

	//optionals
	anounceList []string
	creationDate time.Time
	comment string
	createdBy string
	encoding string

	InfoH Sha1
}

// InfoDict describes the files of the torrent
type InfoDict struct {
	PieceLen int // piece length. number of bytes in each piece
	PiecesHash   []Sha1 //muiltiple of twenty. SHAs of the piece at the corresponding index. byte string
	name string //name of file in single file mode, name of directory in directory mode

	private bool //optional
	isDir bool //specifies whether it is single-file mode or directory



	files []Info // if single-file mode, the slice will contain one item

}

type Info struct {
	length int //length of the file in bytes
	md5sum string
	path string //name of the file if it is a single file. name of the directory if it is a directory
}

func (m MetaInfo) String() string {
	return fmt.Sprintf(
		"Announce: %s\nCreation Time: %s\nCreated By: %s",
		m.Announce, m.creationDate, m.createdBy,
	)
}


func (m *MetaInfo) GetInfoHash() (*Sha1, error) {
	h := sha1.New()
	benc := NewBencoder(h)
	if err := benc.Encode(m.Info); err != nil {
		return nil, err
	}
	sharr := *(*[20]byte)(h.Sum(nil))
	sha := Sha1(sharr)
	m.InfoH = sha
	return &sha, nil
}

func (m MetaInfo) Size() int {
	return 0
}