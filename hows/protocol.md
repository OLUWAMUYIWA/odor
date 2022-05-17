[ref](http://web.cs.ucla.edu/classes/cs217/05BitTorrent.pdf)

P2P is a network architecture where participators (called pers) are equal in ability, and any member(peer) can initiate communication with another. A p2p netwwork may be pure or hybrid. The `pure` case is obvious, its an egalitarian place. The hybrid case is a little different, there exists a member whose duties are central to the network, but the characteristic role of a server in a client-server architecture is not replicated here. The central entity is needed to provide only some(one?) the services in the networ. BitTorrent, for instance is a hybrid p2p arch because it needs the tracker. But the tracker does not do anything beyond peer discovery.

###  Properties of BitTorrent
- BitTorrent uses multiple parallel connections for download
- Peer selection helps select peers who are willing to share files with the client requesting. It uses tthe choke/unchoke mechnaism, the ides of Bitfields, and Have messages


### Protocol
- file owner creates a torrent, describing the file and what is needed to download it. it points to the tracker. it has a `.tottent` ext.
- file owner uploads it to the torrent file to the torrent server
- the file owner is now a seeder, because they have a complete set of the file. they seed the tracker first
- a potential peer can search using google for the torrent file. then they can locate the torrent file (a small file) on the torrent server, and download it
- a peer can then talk to a tracker for a list of peers
- a participating peer then does this: it contacts seeder for pieces, and also trades pieces with other participating peers

Esentially, bittorrent needs the following: 

    - a ‘tracker’
    - a client (us) who is also a leecher
    - a metainfo file. its static and usually publicly available on a bitTorrent server but can be made available through other means, e.g. mail
    - a tracker
    - an original downloader (seed)
    

#### MetaInfo
Its a static file. its `bencoded`. it must contain the address of the tracker, the name of the file,, size, and piece hashes for validatin each downloaded piece

#### Peer
A peer refers to a participating node in a downloa. A peer could be a leecher or a seeder. The fact that BitTorrent nodes consists of peers makes it a p2p protocol. The `Tracker` is the node that trumps them all, it is the central node, and not a peer

#### Tracker
Trackers exist for peer discovery, morally. Trackers dont have the file to be downloaded. It keeps a list of peers that are currently
downloading a file. Thislist of peers are constantly being updated. The list of `peer`s contained in the tracker is called a `swarm`. Tracker and clients communicate using either utp or http.

#### Leecher
A leecher is a peer who does not yet have the complete set of the file. the leecher communicates with the `tracker`, requesting for the list of peers. It downloads `piece`s from the peers, and simultaneously makes available its already downloaded pieces to other leechers. Each piece is verified against its `Sha1` hash which is already in the `MetaInfo` file. A leecher does not need to become a seeder before it starts making its pieces availavle for download.

#### Client
A client refers to `us`. a client is a peer. 

#### Seeder
Seeders are peers too, but they have the complete file. A leecher becomes a seeder when it has fully downloaded the whole file


