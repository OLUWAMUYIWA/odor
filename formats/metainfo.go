package formats

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

type Peer struct {
	IP net.IP
	port uint16
}

func (m *MetaInfo) GetPeers(port uint16) ([]Peer, error) {
	ctx := context.TODO()
	rand.Seed(time.Now().Unix())
	peerId :=  make([]byte, 20, 20)
	rand.Read(peerId)

	baseUrl, err := url.Parse(m.Announce)
	if err != nil {
		return nil, err
	}
	baseUrl.RawQuery = url.Values{
		"info_hash": []string{string(m.infoHash[:])},
		"peer_id":[]string{string(peerId)},
		"port": []string{strconv.FormatInt(6881, 10)},
		"uploaded": []string{"0"},
		"downloaded": []string{"0"},
		"left": []string{strconv.FormatInt(int64(m.info.files[0].length), 10)}, // comeback
		"compact": []string{"1"},
	}.Encode()

	client := http.Client{
		Timeout: 20 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", baseUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	
}