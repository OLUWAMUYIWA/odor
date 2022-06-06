package main

import (
	"crypto/rand"

	"sync"

	"github.com/OLUWAMUYIWA/odor/formats"
)

type Torrent struct {
	mInfo formats.MetaInfo
	peers []PeerAddr
	pl    int
	name  string
}

var once sync.Once
var peerId [20]byte

func (t *Torrent) Start() {
	// get peerID once
	getPerID := func() {
		_, err := rand.Read(peerId[:])
		if err != nil {
			panic("error while creating random peerId: " + err.Error())
		}
	}
	once.Do(getPerID)

}
