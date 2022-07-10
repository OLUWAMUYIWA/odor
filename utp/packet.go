package utp

import "fmt"



const HeaderSize uint = 20;




type PacketType uint8;

const (
	Data PacketType = iota  // data payload
    Fin   // sent as the end of a connection
    State // state of a packet, acked?
    Reset // forcibly ends a connection
    Syn   // initiates a new connection with a remote socket
    Invalid
)

func packetType(n uint8) (PacketType, error) {
	if n == uint8(Data) {
		return Data, nil
	} else if n == uint8(Fin) {
		return Fin, nil
	} else if n == uint8(State) {
		return State, nil
	} else if n == uint8(Reset) {
		return Reset, nil
	} else if n == uint8(Syn) {
		return Syn, nil
	} else {
		return Invalid, fmt.Errorf("Invalid package type: %d", n)
	}
}



func packetByte(p PacketType) uint8 {
	return uint8(p)
}

// only one extension type exists et
type ExtType uint8;
const (
	None ExtType = iota
	SelectiveAck
	InvalidExt
)


func extType(n uint8) (ExtType, error) {
	if n == 0 {
		return None, nil
	} else if n == 1 {
		return SelectiveAck, nil
	} else {
		return InvalidExt, fmt.Errorf("Invalid ext type: %d", n)
	}
}

func extByte(e ExtType) uint8 {
	return uint8(e)
}


type Ext struct {
	data []byte
	ty ExtType
}

func (e Ext) len() int {
	return len(e.data)
}

// etype returns the extension type of the extension
func (e Ext) etype () ExtType {
	ty, _ := extType(uint8(e.ty))
	return ty
}

func (e Ext) Iter() BitStream {
	return NewBitStream(e.data)
}


type PacketHeader struct {
	type_ver uint8 // type: u4, ver: u4
    extension uint8
    connection_id uint16
    
    // Both timestamps are in microseconds
    timestamp uint32
    timestamp_difference uint32

    wnd_size uint32
    seq_nr uint16
    ack_nr uint16
}


