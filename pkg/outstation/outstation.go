package outstation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/internal/logger"
	"github.com/lihongjie0209/dnp3-go/pkg/link"
	"github.com/lihongjie0209/dnp3-go/pkg/transport"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

var (
	ErrOutstationDisabled = errors.New("outstation is disabled")
)

// outstation implements the Outstation interface
type outstation struct {
	config    OutstationConfig
	callbacks OutstationCallbacks
	logger    logger.Logger

	// Data storage
	database    *Database
	eventBuffer *EventBuffer

	// Session
	session *session

	// State
	enabled         bool
	seqCounter      *app.SequenceCounter
	unsolicitedMask app.ClassField // Classes enabled for unsolicited responses
	stateMu         sync.RWMutex

	// Concurrency
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	updateChan chan *updateRequest
}

// session connects the outstation to a channel
type session struct {
	linkAddress uint16
	remoteAddr  uint16
	channel     *channel.Channel
	outstation  *outstation
	transport   *transport.Layer
}

// updateRequest represents an update request
type updateRequest struct {
	builder *UpdateBuilder
	resp    chan error
}

// New creates a new outstation
func New(config OutstationConfig, callbacks OutstationCallbacks, ch *channel.Channel, log logger.Logger) (*outstation, error) {
	if log == nil {
		log = logger.NewNoOpLogger()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create event buffer
	maxEvents := config.MaxBinaryEvents
	if config.MaxAnalogEvents > maxEvents {
		maxEvents = config.MaxAnalogEvents
	}
	eventBuffer := NewEventBuffer(maxEvents)

	// Create database
	database := NewDatabase(config.Database, eventBuffer)

	o := &outstation{
		config:          config,
		callbacks:       callbacks,
		logger:          log,
		database:        database,
		eventBuffer:     eventBuffer,
		enabled:         false,
		seqCounter:      app.NewSequenceCounter(),
		unsolicitedMask: app.ClassAll, // Start with all classes enabled
		ctx:             ctx,
		cancel:          cancel,
		updateChan:      make(chan *updateRequest, 100),
	}

	// Create session
	o.session = &session{
		linkAddress: config.LocalAddress,
		remoteAddr:  config.RemoteAddress,
		channel:     ch,
		outstation:  o,
		transport:   transport.NewLayer(),
	}

	// Add session to channel
	if err := ch.AddSession(o.session); err != nil {
		cancel()
		return nil, err
	}

	o.logger.Info("Outstation %s created: local=%d, remote=%d", config.ID, config.LocalAddress, config.RemoteAddress)
	return o, nil
}

// Enable enables the outstation
func (o *outstation) Enable() error {
	o.stateMu.Lock()
	if o.enabled {
		o.stateMu.Unlock()
		return nil
	}
	o.enabled = true
	o.stateMu.Unlock()

	o.logger.Info("Outstation %s enabled", o.config.ID)

	// Start update processor
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.updateProcessor()
	}()

	// Start unsolicited processor if enabled
	if o.config.AllowUnsolicited {
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()
			o.unsolicitedProcessor()
		}()
	}

	return nil
}

// Disable disables the outstation
func (o *outstation) Disable() error {
	o.stateMu.Lock()
	o.enabled = false
	o.stateMu.Unlock()

	o.logger.Info("Outstation %s disabled", o.config.ID)
	return nil
}

// Shutdown shuts down the outstation
func (o *outstation) Shutdown() error {
	o.logger.Info("Outstation %s shutting down", o.config.ID)

	o.Disable()
	o.cancel()
	o.wg.Wait()

	o.logger.Info("Outstation %s shutdown complete", o.config.ID)
	return nil
}

// Apply applies measurement updates atomically
func (o *outstation) Apply(updates *Updates) error {
	if !o.isEnabled() {
		return ErrOutstationDisabled
	}

	// For now, create a simple builder
	builder := NewUpdateBuilder()

	req := &updateRequest{
		builder: builder,
		resp:    make(chan error, 1),
	}

	select {
	case o.updateChan <- req:
		return <-req.resp
	case <-time.After(1 * time.Second):
		return errors.New("update queue full")
	case <-o.ctx.Done():
		return o.ctx.Err()
	}
}

// SetConfig updates the outstation configuration
func (o *outstation) SetConfig(config OutstationConfig) error {
	o.stateMu.Lock()
	defer o.stateMu.Unlock()

	o.config = config
	return nil
}

// updateProcessor processes measurement updates
func (o *outstation) updateProcessor() {
	for {
		select {
		case <-o.ctx.Done():
			return
		case req := <-o.updateChan:
			o.applyUpdates(req.builder)
			req.resp <- nil
		}
	}
}

// applyUpdates applies updates to the database
func (o *outstation) applyUpdates(builder *UpdateBuilder) {
	updates := builder.GetUpdates()

	for key, value := range updates {
		switch key.pointType {
		case MeasurementTypeBinary:
			if val, ok := value.measurement.(types.Binary); ok {
				o.database.UpdateBinary(key.index, val, value.mode)
			}
		case MeasurementTypeAnalog:
			if val, ok := value.measurement.(types.Analog); ok {
				o.database.UpdateAnalog(key.index, val, value.mode)
			}
		case MeasurementTypeCounter:
			if val, ok := value.measurement.(types.Counter); ok {
				o.database.UpdateCounter(key.index, val, value.mode)
			}
		}
	}
}

// unsolicitedProcessor sends unsolicited responses periodically
func (o *outstation) unsolicitedProcessor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			if o.eventBuffer.HasEvents() {
				o.sendUnsolicitedResponse()
			}
		}
	}
}

// sendUnsolicitedResponse sends an unsolicited response
func (o *outstation) sendUnsolicitedResponse() {
	// TODO: Build and send unsolicited response
	o.logger.Debug("Outstation %s: Would send unsolicited response", o.config.ID)
}

// isEnabled returns true if outstation is enabled
func (o *outstation) isEnabled() bool {
	o.stateMu.RLock()
	defer o.stateMu.RUnlock()
	return o.enabled
}

// getNextSequence returns the next sequence number using app layer helper
func (o *outstation) getNextSequence() uint8 {
	return o.seqCounter.Next()
}

// Session returns the session (for channel registration)
func (o *outstation) Session() channel.Session {
	return o.session
}

// String returns string representation
func (o *outstation) String() string {
	return fmt.Sprintf("Outstation{ID=%s, Local=%d, Remote=%d}",
		o.config.ID, o.config.LocalAddress, o.config.RemoteAddress)
}

// session methods

// OnReceive handles received link frames (implements channel.Session)
func (s *session) OnReceive(frame *link.Frame) error {
	s.outstation.logger.Debug("Outstation session %d: Received frame from %d, FC=%d", s.linkAddress, frame.Source, frame.FunctionCode)

	// Handle link layer control frames
	if frame.IsPrimary == link.PrimaryFrame {
		switch frame.FunctionCode {
		case link.FuncResetLink:
			// Respond with ACK for Reset Link States
			s.outstation.logger.Debug("Outstation session %d: Received Reset Link States, sending ACK", s.linkAddress)
			return s.sendLinkAck()

		case link.FuncRequestLinkStatus:
			// Respond with Link Status
			s.outstation.logger.Debug("Outstation session %d: Received Request Link Status", s.linkAddress)
			return s.sendLinkStatus()

		case link.FuncUserDataConfirmed:
			// For confirmed user data, send ACK first
			s.outstation.logger.Debug("Outstation session %d: Received Confirmed User Data", s.linkAddress)
			if err := s.sendLinkAck(); err != nil {
				return err
			}
			// Then process the data

		case link.FuncUserDataUnconfirmed:
			// Process unconfirmed user data (no ACK needed at link layer)
			s.outstation.logger.Debug("Outstation session %d: Received Unconfirmed User Data", s.linkAddress)
			// Fall through to process APDU
		}
	}

	// If no user data, we're done
	if len(frame.UserData) == 0 {
		return nil
	}

	// Process through transport layer
	apdu, err := s.transport.Receive(frame.UserData)
	if err != nil {
		// Only log critical errors (buffer overflow)
		// Sequence errors and missing FIR are now handled silently by auto-recovery
		s.outstation.logger.Debug("Outstation session %d: Transport error: %v", s.linkAddress, err)
		return nil // Don't propagate error, let transport layer recover
	}

	if apdu == nil {
		// Not complete yet, waiting for more segments (or discarded out-of-sync segment)
		return nil
	}

	// Process complete APDU
	return s.outstation.onReceiveAPDU(apdu)
}

// LinkAddress returns the link address (implements channel.Session)
func (s *session) LinkAddress() uint16 {
	return s.linkAddress
}

// Type returns the session type (implements channel.Session)
func (s *session) Type() channel.SessionType {
	return channel.SessionTypeOutstation
}

// OnConnectionEstablished resets transport layer when connection is established (implements channel.SessionWithConnectionState)
func (s *session) OnConnectionEstablished() {
	s.outstation.logger.Info("Outstation session %d: Connection established, resetting transport layer", s.linkAddress)
	s.transport.Reset()
}

// OnConnectionLost handles connection loss (implements channel.SessionWithConnectionState)
func (s *session) OnConnectionLost() {
	s.outstation.logger.Info("Outstation session %d: Connection lost", s.linkAddress)
	s.transport.Reset()
}

// sendLinkAck sends a link layer ACK response
func (s *session) sendLinkAck() error {
	s.outstation.logger.Debug("Outstation session %d: Sending ACK to %d", s.linkAddress, s.remoteAddr)

	frame := link.NewFrame(
		link.DirectionOutstationToMaster,
		link.SecondaryFrame,
		link.FuncAck,
		s.remoteAddr,
		s.linkAddress,
		nil, // No user data
	)

	data, err := frame.Serialize()
	if err != nil {
		return err
	}

	return s.channel.Write(data)
}

// sendLinkStatus sends a link status response
func (s *session) sendLinkStatus() error {
	s.outstation.logger.Debug("Outstation session %d: Sending Link Status to %d", s.linkAddress, s.remoteAddr)

	frame := link.NewFrame(
		link.DirectionOutstationToMaster,
		link.SecondaryFrame,
		link.FuncLinkStatusResponse,
		s.remoteAddr,
		s.linkAddress,
		nil, // No user data
	)

	data, err := frame.Serialize()
	if err != nil {
		return err
	}

	return s.channel.Write(data)
}

// sendAPDU sends an APDU through the channel
func (s *session) sendAPDU(apdu []byte) error {
	// Segment through transport layer
	segments := s.transport.Send(apdu)

	// Send each segment as a link frame
	for _, segment := range segments {
		frame := link.NewFrame(
			link.DirectionOutstationToMaster,
			link.SecondaryFrame,
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

// onReceiveAPDU handles received APDU
func (o *outstation) onReceiveAPDU(data []byte) error {
	apdu, err := app.Parse(data)
	if err != nil {
		o.logger.Error("Outstation %s: APDU parse error: %v", o.config.ID, err)
		return err
	}

	o.logger.Debug("Outstation %s: Received APDU: %s", o.config.ID, apdu)

	// Process based on function code
	switch apdu.FunctionCode {
	case app.FuncRead:
		return o.handleRead(apdu)
	case app.FuncWrite:
		return o.handleWrite(apdu)
	case app.FuncSelect:
		return o.handleSelect(apdu)
	case app.FuncOperate:
		return o.handleOperate(apdu)
	case app.FuncDirectOperate:
		return o.handleDirectOperate(apdu)
	case app.FuncEnableUnsolicited:
		return o.handleEnableUnsolicited(apdu)
	case app.FuncDisableUnsolicited:
		return o.handleDisableUnsolicited(apdu)
	default:
		o.logger.Warn("Outstation %s: Unsupported function: %s", o.config.ID, apdu.FunctionCode)
		return o.sendErrorResponse(apdu.Sequence)
	}
}

// handleRead handles READ requests
func (o *outstation) handleRead(apdu *app.APDU) error {
	o.logger.Debug("Outstation %s: Handling READ request", o.config.ID)

	// Build response with IIN
	iin := o.callbacks.GetApplicationIIN()

	// Build response data from database
	responseData := o.buildReadResponse(apdu.Objects)

	response := app.NewResponseAPDU(apdu.Sequence, iin, responseData)
	return o.session.sendAPDU(response.Serialize())
}

// handleSelect handles SELECT requests
func (o *outstation) handleSelect(apdu *app.APDU) error {
	o.logger.Debug("Outstation %s: Handling SELECT request", o.config.ID)

	// TODO: Process SELECT through command handler
	iin := o.callbacks.GetApplicationIIN()
	response := app.NewResponseAPDU(apdu.Sequence, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}

// handleOperate handles OPERATE requests
func (o *outstation) handleOperate(apdu *app.APDU) error {
	o.logger.Debug("Outstation %s: Handling OPERATE request", o.config.ID)

	// TODO: Process OPERATE through command handler
	iin := o.callbacks.GetApplicationIIN()
	response := app.NewResponseAPDU(apdu.Sequence, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}

// handleDirectOperate handles DIRECT OPERATE requests
func (o *outstation) handleDirectOperate(apdu *app.APDU) error {
	o.logger.Debug("Outstation %s: Handling DIRECT OPERATE request", o.config.ID)

	// TODO: Process DIRECT OPERATE through command handler
	iin := o.callbacks.GetApplicationIIN()
	response := app.NewResponseAPDU(apdu.Sequence, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}

// handleEnableUnsolicited handles ENABLE UNSOLICITED requests
func (o *outstation) handleEnableUnsolicited(apdu *app.APDU) error {
	o.logger.Debug("Outstation %s: Handling ENABLE UNSOLICITED request", o.config.ID)

	// Parse object headers to determine which classes to enable
	classesToEnable := o.parseClassMask(apdu.Objects)

	o.stateMu.Lock()
	o.unsolicitedMask |= classesToEnable
	o.stateMu.Unlock()

	o.logger.Info("Outstation %s: Enabled unsolicited for classes: %s", o.config.ID, classesToEnable)

	// Send empty response with IIN
	iin := o.callbacks.GetApplicationIIN()
	response := app.NewResponseAPDU(apdu.Sequence, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}

// handleDisableUnsolicited handles DISABLE UNSOLICITED requests
func (o *outstation) handleDisableUnsolicited(apdu *app.APDU) error {
	o.logger.Debug("Outstation %s: Handling DISABLE UNSOLICITED request", o.config.ID)

	// Parse object headers to determine which classes to disable
	classesToDisable := o.parseClassMask(apdu.Objects)

	o.stateMu.Lock()
	o.unsolicitedMask &^= classesToDisable
	o.stateMu.Unlock()

	o.logger.Info("Outstation %s: Disabled unsolicited for classes: %s", o.config.ID, classesToDisable)

	// Send empty response with IIN
	iin := o.callbacks.GetApplicationIIN()
	response := app.NewResponseAPDU(apdu.Sequence, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}

// parseClassMask parses object headers to extract class mask
func (o *outstation) parseClassMask(objects []byte) app.ClassField {
	var mask app.ClassField

	parser := app.NewParser(objects)
	for parser.HasMore() {
		header, err := parser.ReadObjectHeader()
		if err != nil {
			o.logger.Warn("Outstation %s: Failed to parse object header: %v", o.config.ID, err)
			break
		}

		// Map object groups to class fields
		switch header.Group {
		case app.GroupClass0Data:
			mask |= app.Class0
		case app.GroupClass1Data:
			mask |= app.Class1
		case app.GroupClass2Data:
			mask |= app.Class2
		case app.GroupClass3Data:
			mask |= app.Class3
		}
	}

	return mask
}

// sendErrorResponse sends an error response
func (o *outstation) sendErrorResponse(seq uint8) error {
	iin := types.IIN{
		IIN1: 0,
		IIN2: types.IIN2NoFuncCodeSupport,
	}
	response := app.NewResponseAPDU(seq, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}

// handleWrite handles WRITE requests (time sync, IIN control, etc.)
func (o *outstation) handleWrite(apdu *app.APDU) error {
	o.logger.Debug("Outstation %s: Handling WRITE request", o.config.ID)

	parser := app.NewParser(apdu.Objects)

	for parser.HasMore() {
		header, err := parser.ReadObjectHeader()
		if err != nil {
			o.logger.Warn("Outstation %s: Failed to parse WRITE object header: %v", o.config.ID, err)
			break
		}

		switch header.Group {
		case app.GroupTimeDate:
			// Group 50 - Time synchronization
			o.handleTimeSync(header, parser)
		case app.GroupInternalIndications:
			// Group 80 - IIN manipulation (used to clear restart flags)
			o.logger.Debug("Outstation %s: WRITE Group 80 (IIN control) - acknowledged", o.config.ID)
			// Skip the data
			if count, ok := header.Range.(app.CountRange); ok {
				parser.Skip(int(count.Count))
			}
		default:
			o.logger.Debug("Outstation %s: WRITE for unsupported group %d", o.config.ID, header.Group)
		}
	}

	// Send empty acknowledgment
	iin := o.callbacks.GetApplicationIIN()
	response := app.NewResponseAPDU(apdu.Sequence, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}

// handleTimeSync handles time synchronization (Group 50 Variation 1)
func (o *outstation) handleTimeSync(header *app.ObjectHeader, parser *app.Parser) {
	if header.Variation == 1 {
		// Variation 1: Absolute time (6 bytes - 48-bit milliseconds since epoch)
		if data, err := parser.ReadBytes(6); err == nil {
			// DNP3 time is 48-bit milliseconds since Jan 1, 1970 00:00 UTC
			timestamp := uint64(data[0]) |
				uint64(data[1])<<8 |
				uint64(data[2])<<16 |
				uint64(data[3])<<24 |
				uint64(data[4])<<32 |
				uint64(data[5])<<40

			o.logger.Info("Outstation %s: Time synchronized - DNP3 time: %d ms", o.config.ID, timestamp)
			// TODO: Store time offset for timestamping events
		}
	}
}

// buildReadResponse builds response data for READ requests
func (o *outstation) buildReadResponse(requestObjects []byte) []byte {
	if len(requestObjects) == 0 {
		return []byte{}
	}

	parser := app.NewParser(requestObjects)
	var responseData []byte

	for parser.HasMore() {
		header, err := parser.ReadObjectHeader()
		if err != nil {
			o.logger.Warn("Outstation %s: Failed to parse READ object header: %v", o.config.ID, err)
			break
		}

		o.logger.Debug("Outstation %s: READ Group=%d Var=%d Qual=%d",
			o.config.ID, header.Group, header.Variation, header.Qualifier)

		// Handle class reads
		switch header.Group {
		case app.GroupClass0Data:
			// Class 0 - All static data
			responseData = append(responseData, o.buildStaticData()...)
		case app.GroupClass1Data:
			// Class 1 events
			responseData = append(responseData, o.buildEventData(1)...)
		case app.GroupClass2Data:
			// Class 2 events
			responseData = append(responseData, o.buildEventData(2)...)
		case app.GroupClass3Data:
			// Class 3 events
			responseData = append(responseData, o.buildEventData(3)...)
		case app.GroupBinaryInput:
			// Binary input static data
			responseData = append(responseData, o.buildBinaryInputResponse(header)...)
		case app.GroupAnalogInput:
			// Analog input static data
			responseData = append(responseData, o.buildAnalogInputResponse(header)...)
		case app.GroupCounter:
			// Counter static data
			responseData = append(responseData, o.buildCounterResponse(header)...)
		default:
			o.logger.Debug("Outstation %s: Unsupported READ group %d", o.config.ID, header.Group)
		}
	}

	return responseData
}

// buildStaticData builds all static data (Class 0)
func (o *outstation) buildStaticData() []byte {
	var data []byte

	// Build binary inputs
	o.database.mu.RLock()
	if len(o.database.binary) > 0 {
		data = append(data, o.buildBinaryInputResponse(nil)...)
	}
	// Build analog inputs
	if len(o.database.analog) > 0 {
		data = append(data, o.buildAnalogInputResponse(nil)...)
	}
	// Build counters
	if len(o.database.counter) > 0 {
		data = append(data, o.buildCounterResponse(nil)...)
	}
	o.database.mu.RUnlock()

	return data
}

// buildEventData builds event data for specified class
func (o *outstation) buildEventData(class uint8) []byte {
	// TODO: Implement event data building from event buffer
	// For now, return empty (NULL response = no events)
	o.logger.Debug("Outstation %s: Building event data for class %d (not yet implemented)", o.config.ID, class)
	return []byte{}
}

// buildBinaryInputResponse builds binary input response using app layer helpers
func (o *outstation) buildBinaryInputResponse(header *app.ObjectHeader) []byte {
	o.database.mu.RLock()
	defer o.database.mu.RUnlock()

	if len(o.database.binary) == 0 {
		return []byte{}
	}

	// Determine variation to use
	variation := uint8(app.BinaryInputWithFlags) // Default to variation 2 (with flags)
	if header != nil && header.Variation != app.VariationAny {
		variation = header.Variation
	} else if len(o.database.binary) > 0 {
		variation = o.database.binary[0].staticVariation
	}

	// Use app layer builder
	builder := app.NewObjectBuilder()

	// Add object header
	builder.AddHeader(
		app.GroupBinaryInput,
		variation,
		app.Qualifier8BitStartStop,
		app.StartStopRange{Start: 0, Stop: uint32(len(o.database.binary) - 1)},
	)

	// Serialize values using app layer helper
	for _, point := range o.database.binary {
		bi := app.BinaryInput{
			Value: point.value.Value,
			Flags: uint8(point.value.Flags),
		}
		builder.AddRawData(bi.Serialize())
	}

	return builder.Build()
}

// buildAnalogInputResponse builds analog input response using app layer helpers
func (o *outstation) buildAnalogInputResponse(header *app.ObjectHeader) []byte {
	o.database.mu.RLock()
	defer o.database.mu.RUnlock()

	if len(o.database.analog) == 0 {
		return []byte{}
	}

	// Determine variation to use
	variation := uint8(app.AnalogInputFloat) // Default to variation 5 (float)
	if header != nil && header.Variation != app.VariationAny {
		variation = header.Variation
	} else if len(o.database.analog) > 0 {
		variation = o.database.analog[0].staticVariation
	}

	// Use app layer builder
	builder := app.NewObjectBuilder()

	// Add object header
	builder.AddHeader(
		app.GroupAnalogInput,
		variation,
		app.Qualifier8BitStartStop,
		app.StartStopRange{Start: 0, Stop: uint32(len(o.database.analog) - 1)},
	)

	// Serialize values using app layer helper
	for _, point := range o.database.analog {
		ai := app.AnalogInput{
			Value: point.value.Value,
			Flags: uint8(point.value.Flags),
		}

		var serialized []byte
		switch variation {
		case app.AnalogInputFloat:
			serialized = ai.SerializeFloat()
		case app.AnalogInput32Bit:
			serialized = ai.Serialize32Bit()
		case app.AnalogInput16Bit:
			serialized = ai.Serialize16Bit()
		case app.AnalogInputDouble:
			serialized = ai.SerializeDouble()
		default:
			serialized = ai.SerializeFloat()
		}
		builder.AddRawData(serialized)
	}

	return builder.Build()
}

// buildCounterResponse builds counter response using app layer helpers
func (o *outstation) buildCounterResponse(header *app.ObjectHeader) []byte {
	o.database.mu.RLock()
	defer o.database.mu.RUnlock()

	if len(o.database.counter) == 0 {
		return []byte{}
	}

	// Determine variation
	variation := uint8(app.Counter32BitWithFlag) // Default to variation 5
	if header != nil && header.Variation != app.VariationAny {
		variation = header.Variation
	} else if len(o.database.counter) > 0 {
		variation = o.database.counter[0].staticVariation
	}

	// Use app layer builder
	builder := app.NewObjectBuilder()

	// Add object header
	builder.AddHeader(
		app.GroupCounter,
		variation,
		app.Qualifier8BitStartStop,
		app.StartStopRange{Start: 0, Stop: uint32(len(o.database.counter) - 1)},
	)

	// Serialize values using app layer helper
	for _, point := range o.database.counter {
		counter := app.Counter{
			Value: point.value.Value,
			Flags: uint8(point.value.Flags),
		}

		var serialized []byte
		switch variation {
		case app.Counter32BitWithFlag, app.Counter32Bit:
			serialized = counter.Serialize32Bit()
		case app.Counter16BitWithFlag, app.Counter16Bit:
			serialized = counter.Serialize16Bit()
		default:
			serialized = counter.Serialize32Bit()
		}
		builder.AddRawData(serialized)
	}

	return builder.Build()
}
