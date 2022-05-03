package formats

import (
	"crypto/sha1"
	"fmt"
	"time"
)

// https://wiki.theory.org/index.php/BitTorrentSpecification

type MetaInfo struct {
	info Info
	Announce string // url of the tracker

	//optionals
	anounceList []string
	creationDate time.Time
	comment string
	createdBy string
	encoding string

	infoHash Sha1
}

type Info struct {
	pl int // piece length. number of bytes in each piece
	pieces   []Sha1 //muiltiple of twenty. SHAs of the piece at the corresponding index

	private bool //optional
	isDir bool //specifies whether it is single-file mode or directory

	name string //name of file in single file mode, name of directory in directory mode


	files []Sub // if single-file mode, the slice will contain one item

}

type Sub struct {
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
	if err := benc.Encode(m.info); err != nil {
		return nil, err
	}
	sharr := *(*[20]byte)(h.Sum(nil))
	sha := Sha1(sharr)
	m.infoHash = sha
	return &sha, nil
}

func (m MetaInfo) Size() int {
	
}