package main

import "github.com/OLUWAMUYIWA/odor/formats"


type Piece struct {
	index int
	hash formats.Sha1
	len int
}

// 2 ^ 14
const BLOCK_LEN int = 16384

