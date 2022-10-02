package formats

import (
	"crypto/sha1"
	"fmt"
	"time"
)

// https://wiki.theory.org/index.php/BitTorrentSpecification

// 2 ^ 14
const BLOCK_LEN int = 16384

type MetaInfo struct {
	Info     InfoDict
	Announce string // url of the tracker

	//optionals
	AnounceList  []string
	CreationDate time.Time
	Comment      string
	CreatedBy    string
	Encoding     string
}

// InfoDict describes the files of the torrent
type InfoDict struct {
	PieceLen   int    // piece length. number of bytes in each piece
	PiecesHash []Sha1 //muiltiple of twenty. SHAs of the piece at the corresponding index. byte string
	Name       string //name of file in single file mode, name of directory in directory mode

	private bool //optional
	isDir   bool //specifies whether it is single-file mode or directory

	Files []Info // if single-file mode, the slice will contain one item

}

type Info struct {
	Length int //length of the file in bytes
	MD5sum string
	Path   string //name of the file if it is a single file. name of the directory if it is a directory
}

func (m MetaInfo) String() string {
	return fmt.Sprintf(
		"Announce: %s\nCreation Time: %s\nCreated By: %s",
		m.Announce, m.CreationDate, m.CreatedBy,
	)
}

func (m MetaInfo) GetInfoHash() (Sha1, error) {
	h := sha1.New()
	benc := NewBencoder(h)
	if err := benc.Encode(m.Info); err != nil {
		return Sha1{}, err
	}
	sharr := *(*[20]byte)(h.Sum(nil))
	sha := Sha1(sharr)
	return sha, nil
}

// Size gives the total size (in bytes) of the torrent, whether its a single file or not
func (m MetaInfo) Size() int {
	var size int
	// comeback to ensure that isDir is set properly during decoding
	if !m.Info.isDir {
		return m.Info.Files[0].Length
	}

	for _, s := range m.Info.Files {
		size += s.Length
	}
	return size
}

// PieceLen gets the length of a piece given its index
func (m MetaInfo) PieceLen(index int) int {
	l := m.Size()
	if l/m.Info.PieceLen == index { // is the index the last one?
		return l % m.Info.PieceLen // the last piece_len may not exactly be the pieceLength specified in the InfoDict
	} else {
		return m.Info.PieceLen
	}
}

func (m MetaInfo) PieceBounds(index int) (int, int) {
	start := index * m.Info.PieceLen
	// normally
	end := start + m.Info.PieceLen
	if end > m.Size() {
		return start, m.Size()
	}
	return start, end
}

// BlockLen gets the length of a specific block in an index, given the piece index and block index
func (m MetaInfo) BlockLen(pIndex, bIndex int) int {

	pLen := m.PieceLen(pIndex) // piece length
	lastBlockLen := pLen % BLOCK_LEN
	if bIndex == pLen/BLOCK_LEN { // if this block is the last, then it might not be full
		return lastBlockLen
	}
	return BLOCK_LEN
}

// NumBlocksInPiece gets the number of blocks in a pice given the index of the piece
func (m MetaInfo) NumBlocksInPiece(index int) int {
	pLen := m.PieceLen(index)
	if pLen%BLOCK_LEN > 0 {
		return (pLen / BLOCK_LEN) + 1
	}
	return pLen / BLOCK_LEN
}
