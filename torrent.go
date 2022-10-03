package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"os"
	"time"

	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/OLUWAMUYIWA/odor/formats"
)

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

type Torrent struct {
	mInfo formats.MetaInfo
	InfoH formats.Sha1 // infohash
	size  int          // size of torrent file in bytes
	peers []PeerAddr
	// pl    int
	name    string
	mu      sync.Mutex
	clients []*PeerConn // list of connections to peers this client is connected to
	fPath   string      // path where to save the torrent
}

func NewTorrent(ctx context.Context, torrPath, fPath string) (*Torrent, error) {
	var t Torrent
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

	t.fPath = fPath

	return &t, nil
}

func (t *Torrent) pieceHashes() []formats.Sha1 {
	return t.mInfo.Info.PiecesHash
}

// Connect connects to a peer and does the handshake, requests bitfields/ haves,
func (t *Torrent) Connect(ctx context.Context, addr PeerAddr) (*PeerConn, error) {
	// create new connection with a peer
	if cl, err := NewConn(ctx, addr); err != nil {
		return nil, err
	} else {
		// handshake with peer with our shaker
		h := NewShaker(t.InfoH, peerId)
		err := cl.Shake(h)
		if err != nil {
			return nil, err
		}

		// et the pieces the peer has
		if err = cl.ReqBitFields(); err != nil {
			return nil, err
		}
		// comeback to check state. suppose it begins with being choked and interested
		cl.state = ChkdIntd
		// now add the client to the client list
		t.clients = append(t.clients, cl)
		return cl, nil
	}
}

type PieceReq struct {
	index  int
	sha    formats.Sha1
	length int
}

type Piece struct {
	index int
	buf   []byte
}

func (t *Torrent) downloadPiece(ctx context.Context, p PeerAddr, pReqChan chan *PieceReq, pChan chan *Piece, errchan chan error) {
	cl, err := t.Connect(ctx, p)
	// comeback send error to errchan, and mayby log the error and close
	if err != nil {
		errchan <- err
	}
	defer cl.conn.Close()

	// first unchoke and make interest known to peer
	if err := cl.Unchoke(); err != nil {
		errchan <- err
	}

	if err := cl.Interested(); err != nil {
		errchan <- err
	}

	for pReq := range pReqChan {
		if !cl.HasPiece(pReq.index) { // put back in chan
			pReqChan <- pReq
			continue
		}

	}

}

func (t *Torrent) Start(ctx context.Context) error {
	// get the number of pieces
	workersNum := len(t.pieceHashes())
	reqChan := make(chan *PieceReq, workersNum)
	pChan := make(chan *Piece)
	errChan := make(chan error)

	// send all to the workers channel to be distributed among clients
	for i, sha := range t.pieceHashes() {
		pLen := t.mInfo.PieceLen(i)
		reqChan <- &PieceReq{index: i, sha: sha, length: pLen}

	}

	for _, peer := range t.peers {
		go t.downloadPiece(ctx, peer, reqChan, pChan, errChan)
	}

	f, err := os.OpenFile(t.fPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	defer f.Close()
	if err != nil {
		return err
	}

	g := new(errgroup.Group)

	for i := 0; i < len(t.pieceHashes()); i++ {
		p := <-pChan
		start, _ := t.mInfo.PieceBounds(p.index)
		if len(p.buf) != t.mInfo.PieceLen(p.index) {
			return fmt.Errorf("Incomplete piece")
		}
		g.Go(func() error {
			_, err := f.WriteAt(p.buf, int64(start))
			return err
		})
	}

	wgErr := g.Wait()
	if wgErr != nil {
		return fmt.Errorf("Could not finish downloading because: %u", wgErr)
	}

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
