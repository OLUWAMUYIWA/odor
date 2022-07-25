package utp

type UTPStream struct {
	sock *UtpSocket
}

func ConnectStream(addr string) (*UTPStream, error) {

	if sock, err := connect(addr); err == nil {
		return &UTPStream{sock: sock}, nil
	} else {
		return nil, err
	}
}

func (s *UTPStream) Close() error {
	return s.sock.Close()
}

func (s *UTPStream) LocalAddr() string {
	return s.sock.localAddr()
}

func (s *UTPStream) SetMaxRetransRetries(n uint32) {
	s.sock.maxRetransmissionRetries = n
}

// impl reader
func (s *UTPStream) Read(p []byte) (int, error) {
	n, _, err := s.sock.RecvFrom(p)
	return n, err
}

// impl writer
func (s *UTPStream) Write(p []byte) (int, error) {
	return s.sock.SendTo(p)
}

func (s *UTPStream) Flush() error {
	return s.sock.Flush()
}
