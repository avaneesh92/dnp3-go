package master

import (
	"avaneesh/dnp3-go/pkg/channel"
	"avaneesh/dnp3-go/pkg/link"
	"avaneesh/dnp3-go/pkg/transport"
)

// session connects the master to a channel
type session struct {
	linkAddress uint16
	remoteAddr  uint16
	channel     *channel.Channel
	master      *master
	transport   *transport.Layer
}

// newSession creates a new master session
func newSession(linkAddr, remoteAddr uint16, ch *channel.Channel, m *master) *session {
	return &session{
		linkAddress: linkAddr,
		remoteAddr:  remoteAddr,
		channel:     ch,
		master:      m,
		transport:   transport.NewLayer(),
	}
}

// OnReceive handles received link frames (implements channel.Session)
func (s *session) OnReceive(frame *link.Frame) error {
	s.master.logger.Debug("Master session %d: Received frame from %d", s.linkAddress, frame.Source)

	// Process through transport layer
	apdu, err := s.transport.Receive(frame.UserData)
	if err != nil {
		s.master.logger.Error("Master session %d: Transport error: %v", s.linkAddress, err)
		return err
	}

	if apdu == nil {
		// Not complete yet, waiting for more segments
		return nil
	}

	// Process complete APDU
	return s.master.onReceiveAPDU(apdu)
}

// LinkAddress returns the link address (implements channel.Session)
func (s *session) LinkAddress() uint16 {
	return s.linkAddress
}

// Type returns the session type (implements channel.Session)
func (s *session) Type() channel.SessionType {
	return channel.SessionTypeMaster
}

// sendAPDU sends an APDU through the channel
func (s *session) sendAPDU(apdu []byte) error {
	// Segment through transport layer
	segments := s.transport.Send(apdu)

	// Send each segment as a link frame
	for _, segment := range segments {
		frame := link.NewFrame(
			link.DirectionMasterToOutstation,
			link.PrimaryFrame,
			link.FuncUserDataUnconfirmed,
			s.remoteAddr,
			s.linkAddress,
			segment,
		)

		data, err := frame.Serialize()
		if err != nil {
			return err
		}

		if err := s.channel.Write(data); err != nil {
			return err
		}
	}

	return nil
}
