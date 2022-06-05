package main

import "github.com/OLUWAMUYIWA/odor/formats"

type Queue struct {
	torrent Torrent
	queue   []formats.Ibl
	chocked bool
}

func NeWQ(t Torrent) Queue {
	return Queue{
		torrent: t,
		queue:   make([]formats.Ibl, 1),
		chocked: true,
	}
}

// takes the piece index
func (q *Queue) enq(index int) {
	numBlocks := q.torrent.mInfo.NumBlocksInPiece(index)
	for i := 0; i < numBlocks; i++ {
		q.queue = append(q.queue, formats.Ibl{
			Index:  index,
			Begin:  i * formats.BLOCK_LEN,
			Length: q.torrent.mInfo.BlockLen(index, i),
		})
	}
}

func (q *Queue) deq() formats.Ibl {
	ret := q.queue[0]
	q.queue = q.queue[1:]
	return ret
}

func (q *Queue) peek() formats.Ibl {
	return q.queue[0]
}

func (q *Queue) len() int {
	return len(q.queue)
}
