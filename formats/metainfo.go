package formats

import (
	"fmt"
	"time"
)

// https://wiki.theory.org/index.php/BitTorrentSpecification

type MetaInfo struct {
	info Info
	anounce string // url of the tracker

	//optionals
	anounceList []string
	creationDate time.Time
	comment string
	createdBy string
	encoding string
}

type Info struct {
	pl int // piece length
	pieces   []Sha1 //muiltiple of twenty. SHAs of the piece at the corresponding index
	len      int        //length of the file in bytes
	path     []byte     //name of the file if it is a single file. name of the directory if it is a directory

	private bool //optional
	isDir bool //specifies whether it is single-file mode or directory

	name string //name of file in single file mode, name of directory in directory mode

	length int
	md5sum string

	files []SubFile

}

type SubFile struct {
	length int
	md5sum string
	path string
}

func (m MetaInfo) String() string {
	return fmt.Sprintf(
		"Announce: %s\nCreation Time: %s\nCreated By: %s",
		m.anounce, m.creationDate, m.createdBy,
	)
}
