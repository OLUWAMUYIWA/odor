package main

import (
	"context"
	"crypto/rand"
	"os"

	"sync"

	"github.com/OLUWAMUYIWA/odor/formats"
)

type Torrent struct {
	mInfo formats.MetaInfo
	InfoH formats.Sha1 // infohash
	size  int          // size of torrent file in bytes
	peers []PeerAddr
	pl    int
	name  string
}

var once sync.Once
var peerId [20]byte

func Init() {
	// get peerID OAFA
	getPerID := func() {
		_, err := rand.Read(peerId[:])
		if err != nil {
			panic("error while creating random peerId: " + err.Error())
		}
	}
	once.Do(getPerID)
}

func NewTorrent(path string) (*Torrent, error) {
	var t Torrent
	ctx := context.TODO()
	// open torrent file
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// decode torrent file into MetaInfo var
	bDec := formats.NewBencDecoder(file)
	var mInfo formats.MetaInfo
	if err := bDec.Decode(&mInfo); err != nil {
		return nil, err
	}
	t.mInfo = mInfo

	// get infohash
	t.InfoH, err = mInfo.GetInfoHash()

	// get torrent size
	t.size = mInfo.Size()

	// get peers using a UDPT client.... UDPT means UDP tracker protocol
	annResp, err := GetPeers(ctx, t)
	if err != nil {
		return nil, err
	}
	t.peers = annResp.socks

	// now o and download
	return &t, nil

}
func (t *Torrent) Start(path string) error {

	return nil
}
