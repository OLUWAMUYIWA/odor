- Piece: A `piece` is a verifiable unit of data. It's described in the `MetaInfo` file. It is verifiable by a `Sha1` hash
- Block: Smallest unit of requestible data from a peer. `Blocks` make up `Pieces`, Two or more
- Peer: Any agent participating in the download
- Client: The user-agent that is the local macjine. The `client` is also a `peer`

## MetaInfo File
- It is _**bencoded**_.
- It is a _dictionary_.
- The filename ends with `.torrent` extension.

### Contents of the MetaInfo

- **info**: a dictionary. it describes the files in the torrent. Two types: single file-types and directory-type
- **announce**: a string representing the address url of the tracker
- creation-date
- comment
- created-by
- encoding: string specifying the used in generating the `pieces` in the `info` dictionary

### Contents of the Ifo Dictionary

- **piece-length**: `bencode integer` representing the number of bytes in each `piece`
- **pieces**: string. concatenation of all the `Sha1` values for each `piece`. Byte strings, not hex-encoded
- private: optional

Now to the parts of the `info` dictionary that depend on wheter its a single-file or directory info

#### Single-file info
- **name**: filename, string
- **length**: file length in bytes
- md5sum: 	for verification. nnot used by bittorrent

#### Directory/ Multi-file Mode
- **name**: directory name
- **files**: `bencode list` of `bencode dicts`. each dict has: 
	- **length**: file length in bytes
	- md5sum: useless to bittorrent
	- **path**: `bencode` `list` of `bencode` `strings`, each representing the sequence of directories until the last one being the filename



## Tracker HTTP/HTTPS Protocol

The `tracker` service responds to `get` requests. `req` sent to the tracker includes values that help the tracker keep track of statistics about the torrent. the `resp` from a tracker service is a `peer` list. 
Tracker req details: 
	- **base_url**: announce url in metainfo file
	- then, a `?` indicating the beginnig of query parameters
	- **info_hash**: `hash` of the `info` dictionary. url-encoded
	- **peer_id**: url-encoded 20-byte string representing the client. could be any byte string
	- **port**: client port. bittorrent reserved: 6881-6889
	- **uploaded**: total amount uploaded since `started` event was sent to the tracker. in base-10 ascii.   
	- **downloaded**: total amount downloaded since `started` event was sent to the tracker. in base-10 ascii.  