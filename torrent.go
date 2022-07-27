package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"math/rand"
	"os"
	"time"

	"sync"

	"github.com/OLUWAMUYIWA/odor/formats"
)

type Torrent struct {
	mInfo formats.MetaInfo
	InfoH formats.Sha1 // infohash
	size  int          // size of torrent file in bytes
	peers []PeerAddr
	// pl    int
	name    string
	mu      sync.Mutex
	clients []*PeerConn // list of connections to peers this client is connected to
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
		// seed the random number enerator too
		rand.Seed(time.Now().Unix())
	}
	once.Do(getPerID)
}

func NewTorrent(torrPath, fPath string) (*Torrent, error) {
	var t Torrent
	ctx := context.TODO()
	// open torrent file
	file, err := os.OpenFile(torrPath, os.O_RDONLY, 0)
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
	annResp, err := GetPeers(ctx, &t)
	if err != nil {
		return nil, err
	}
	t.peers = annResp.socks

	return &t, nil

}

// Connect connects to a peer and does the handshake, requests bitfields/ haves,
func (t *Torrent) Connect(ctx context.Context, addr PeerAddr) error {
	// create new connection with a peer
	if cl, err := NewConn(ctx, addr); err != nil {
		return err
	} else {
		// handshake with peer
		h := NewHandShake(t.InfoH, peerId)
		err := cl.Shake(h)
		if err != nil {
			return err
		}

		// et the pieces the peer has
		if err = cl.ReqBitFields(); err != nil {
			return err
		}
		// comeback to check state. suppose it begins with being choked and interested
		cl.state = ChkdIntd
		// now add the client to the client list
		t.clients = append(t.clients, cl)
		return nil
	}
}

func (t *Torrent) Start() error {

	return nil
}

// verifyPiece checks if the sha1 hash of a fully downloaded piece is what we expected as compared with the PieceHash in its index
// returns a boolean
func (t *Torrent) verifyPiece(index int, pieceBytes []byte) bool {
	sha := sha1.New()
	sha.Write(pieceBytes)
	hash := sha.Sum(nil)
	if bytes.Compare(hash, t.mInfo.Info.PiecesHash[index][:]) != 0 {
		return false
	}
	return true
}
