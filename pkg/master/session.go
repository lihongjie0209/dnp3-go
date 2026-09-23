package master

import (
	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/link"
	"github.com/lihongjie0209/dnp3-go/pkg/transport"
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
		// Only log critical errors (buffer overflow)
		// Sequence errors and missing FIR are now handled silently by auto-recovery
		s.master.logger.Debug("Master session %d: Transport error: %v", s.linkAddress, err)
		return nil // Don't propagate error, let transport layer recover
	}

	if apdu == nil {
		// Not complete yet, waiting for more segments (or discarded out-of-sync segment)
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

// OnConnectionEstablished resets transport layer when connection is established (implements channel.SessionWithConnectionState)
func (s *session) OnConnectionEstablished() {
	s.master.logger.Info("Master session %d: Connection established, resetting transport layer", s.linkAddress)
	s.transport.Reset()
}

// OnConnectionLost handles connection loss (implements channel.SessionWithConnectionState)
func (s *session) OnConnectionLost() {
	s.master.logger.Info("Master session %d: Connection lost", s.linkAddress)
	s.transport.Reset()
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
