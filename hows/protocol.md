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
A peer refers to a participating node in a torent. A peer could be a leecher or a seeder. WRT you, Peers are other users participating in a torrent, and have the partial file, or the complete file. When they have complete file, they are known as `seed`s. The fact that BitTorrent nodes consists of peers makes it a p2p protocol. The `Tracker` is the node that trumps them all, it is the central node, and not a peer

#### Tracker
Trackers exist for peer discovery, morally. Trackers dont have the file to be downloaded. It keeps a list of peers that are currently downloading a file. This list of peers are constantly being updated. The list of `peer`s contained in the tracker is called a `swarm`. Tracker and clients communicate using either utp or http. The tracker is constantly replying to connecting peers with a list of peers who have the requested pieces.

#### Leecher
A leecher is a peer who does not yet have the complete set of the file. the leecher communicates with the `tracker`, requesting for the list of peers. It downloads `piece`s from the peers, and simultaneously makes available its already downloaded pieces to other leechers. Each piece is verified against its `Sha1` hash which is already in the `MetaInfo` file. A leecher does not need to become a seeder before it starts making its pieces availavle for download.

#### Client
A client refers to `us`. a client is a peer. 

#### Seeder
Seeders are peers too, but they have the complete file. A leecher becomes a seeder when it has fully downloaded the whole file




### Message Types

- Choked: the peer does not wish to share pieces with you
- Unchoked: the peer is willing to share with you
- Iterested: what you send to another peer/ or another peer sends to you to indicate interest in what they/you have
- Uninterested: opposite of `Interested`.  
- Have: The `Have` message allows a peer to specify the piece index of the pieces it has. its payload is the piece index. This means that a peer may have to send more than one message to fully state the piece indexes it has
- BitField: The Bitfield message is a more compact form of the `Have` message. It allows a peer to specify the piece indexes it has by using bit fields. It makes use of a strin of bits. If the peer has it, the bit is set to `1`, else `0`. The string of bits matches the number of pieces beloning to the torrent.
- Request: The `Request` message has as it's payload the piece `index`; the `begin`, which specifies the byte offset within the piece, and the `length` which spcifies the length in bytes of the portion of the piece extending from the `begin`. These three numbers are `uint32`. The `Request` is used to specify a `block` that you want.
- Piece: The `Piece` message is the response to the `Request` message. In it you have as the payload the piece `index`, which is same as in `Request`; the `begin`, same as in index, and the `block`, which represents the real deal we want. It is the block we have requested for.
- cancel: it is used to cancel block requests (what you send using the `Request` message). 


### Process: 
1. You begin with the `HandShake`. If the is ready to communicate with you, they send a similar Handshake message in  return. If tey don't, they might just close the conection.
2. With every peer you shake hands with, you begin by being `choke`d and `uninterested`. So, as a client, you send the `Interested` message to them, and if they send an `Unchoke` message to you, you can download from them. 
3. If they send a `choke` back, close the connection. 
4. Block request comes next. You use the `request`  message, specifying the block (a portion of the piece) you want from them.
5. The peer sends you a `Piece` message in return, containing both the bytes of the block of the file being downloaded, and other info needed to put it in its place (i.e. the piece index and the begin)
6. you may then assemble the blocks received

### Piece Selection
1. Random First Piece: The first strategy. The peer has nothing to upload yet when it begins. So it selects a piece at random for download. The client continuously selects random pieces until the first piece is fully downloaded and checked. After this, it changes strategy to Rarest first
2. Rarest First: At the beginning of the lifespan of a torrent, only one Seed is assumed to exist. If so many peers wre trying to access one single piece, it becomes a bottleneck. To prevent this, Rarest-First strategy suggests that the client asks for the Piece held by the lowest number of peers. Such pieces change as peers request and download pieces and incoming peers make their own request. A piece that is rarest at this moment likely won't be in the next few moments after a sufficient number of peers ask for it. Load becomes better distributed in the system. . Rarest first also works to prevent the possible loss of pieces due to the disconnection/uavailability of seed(s). It does this by replicating the pieces most at risk as quickly as possible. Genius move!
3. EndGame Mode: Near the end of a download, a download might slow down because the lingering downloads are from peers with slow transfer rates. In such a situation, the client switches to EndGame Mode. Here, the remaining sub-pieces are requested from all peers in the current swarm. 