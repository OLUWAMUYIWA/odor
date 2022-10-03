package main

import (
	"github.com/OLUWAMUYIWA/odor/formats"
)

// PiecesState represents the state of all blocks in the torrent
// field Reqd is the slice of states of each block in a piece. it shows whether the block has been requested for or not
// field Recvd is the slice of states of each block in a piece. it shows whether the block has been received for or not
type PiecesState struct {
	Reqd /* requested */, Recvd /* received */ []PState
}

type PState struct {
	done []bool
}

func NewPieces(m formats.MetaInfo) PiecesState {
	numPieces := len(m.Info.PiecesHash) // the pieceshash is a slice representing the hash of each of the pieces
	// make a slice of `PState`s for reqd
	req := make([]PState, numPieces)
	for i := 0; i < numPieces; i++ {
		pieceSlice := make([]bool, m.NumBlocksInPiece(i))
		req[i] = PState{
			pieceSlice,
		}
	}
	// comeback. how do i clone a slice
	rcv := make([]PState, numPieces)
	for i := 0; i < numPieces; i++ {
		pieceSlice := make([]bool, m.NumBlocksInPiece(i))
		rcv[i] = PState{
			pieceSlice,
		}
	}
	return PiecesState{
		Reqd:  req,
		Recvd: rcv,
	}
}

// assertReqd takes a `PieceMsg` and uses it to assert that a particular block is requested
func (p *PiecesState) assertReqd(piece formats.Ibl) {
	p.Reqd[int(piece.Index)].done[int(piece.Begin)/formats.BLOCK_LEN] = true

}

// assertRecvd takes a PieceMsg` and uses it to assert that a particular block is received
func (p *PiecesState) assertRecvd(piece formats.PieceMsg) {
	p.Recvd[int(piece.Index)].done[int(piece.Begin)/formats.BLOCK_LEN] = true
}

func (p *PiecesState) pieceDone(index int) bool {
	pieceRcvd := p.Recvd[index]
	for _, d := range pieceRcvd.done {
		if !d {
			return false
		}
	}
	return true
}

// isDone checks if all blocks in all pieces have been received. returns a boolean indicatin the status
func (p PiecesState) isDone() bool {
	for _, p := range p.Recvd {
		for _, b := range p.done {
			if !b {
				return false
			}
		}
	}
	return true
}

func (p *PiecesState) needed(piece formats.Ibl) bool {
	allReqd := true
	for _, p := range p.Reqd {
		for _, b := range p.done {
			if !b {
				allReqd = false
				break
			}
		}
		if !allReqd {
			break
		}
	}
	if allReqd { // if all blocks have been requested for
		for i := range p.Recvd {
			currRcvd := p.Recvd[i]
			// allocate a new buffer so the two slices won't be linked
			reqd := PState{
				done: make([]bool, len(currRcvd.done)),
			}
			//copy from received to requested
			copy(reqd.done, currRcvd.done)
			p.Reqd[i] = reqd
		}
	}
	return !p.Reqd[piece.Index].done[int(piece.Begin)/formats.BLOCK_LEN]
}
