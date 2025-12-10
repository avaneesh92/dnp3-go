package outstation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"avaneesh/dnp3-go/pkg/app"
	"avaneesh/dnp3-go/pkg/channel"
	"avaneesh/dnp3-go/pkg/internal/logger"
	"avaneesh/dnp3-go/pkg/link"
	"avaneesh/dnp3-go/pkg/transport"
	"avaneesh/dnp3-go/pkg/types"
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
	enabled  bool
	sequence uint8
	stateMu  sync.RWMutex

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
		config:      config,
		callbacks:   callbacks,
		logger:      log,
		database:    database,
		eventBuffer: eventBuffer,
		enabled:     false,
		sequence:    0,
		ctx:         ctx,
		cancel:      cancel,
		updateChan:  make(chan *updateRequest, 100),
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

// getNextSequence returns the next sequence number
func (o *outstation) getNextSequence() uint8 {
	o.stateMu.Lock()
	defer o.stateMu.Unlock()
	seq := o.sequence
	o.sequence = (o.sequence + 1) & 0x0F
	return seq
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
	s.outstation.logger.Debug("Outstation session %d: Received frame from %d", s.linkAddress, frame.Source)

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
	case app.FuncSelect:
		return o.handleSelect(apdu)
	case app.FuncOperate:
		return o.handleOperate(apdu)
	case app.FuncDirectOperate:
		return o.handleDirectOperate(apdu)
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

	// TODO: Build response data from database
	responseData := []byte{}

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

// sendErrorResponse sends an error response
func (o *outstation) sendErrorResponse(seq uint8) error {
	iin := types.IIN{
		IIN1: 0,
		IIN2: types.IIN2NoFuncCodeSupport,
	}
	response := app.NewResponseAPDU(seq, iin, nil)
	return o.session.sendAPDU(response.Serialize())
}
