package link

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OutstationLink implements link layer for DNP3 outstation
type OutstationLink struct {
	// Configuration
	localAddress  uint16
	remoteAddress uint16
	timeout       time.Duration

	// State
	state              LinkState
	fcbValidator       *FCBValidator
	unsolicitedEnabled bool

	// Callbacks
	dataCallback   DataCallback
	statusCallback StatusCallback

	// Channels for communication
	sendChan chan []byte // To physical layer

	// Synchronization
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// FCBValidator handles Frame Count Bit validation and duplicate detection
type FCBValidator struct {
	lastFCB     bool
	initialized bool
	mu          sync.Mutex
}

// NewFCBValidator creates a new FCB validator
func NewFCBValidator() *FCBValidator {
	return &FCBValidator{
		initialized: false,
		lastFCB:     false,
	}
}

// ValidateAndUpdate validates FCB and updates state
// Returns true if frame is a duplicate
func (v *FCBValidator) ValidateAndUpdate(fcb bool, fcv bool) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !fcv {
		// FCB not valid - not using duplicate detection
		return false
	}

	if !v.initialized {
		// First confirmed message
		v.initialized = true
		v.lastFCB = fcb
		return false
	}

	if fcb == v.lastFCB {
		// Duplicate frame
		return true
	}

	// New frame - update stored FCB
	v.lastFCB = fcb
	return false
}

// Reset resets the FCB validator state
func (v *FCBValidator) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.initialized = false
	v.lastFCB = false
}

// NewOutstationLink creates a new outstation link layer
func NewOutstationLink(config LinkLayerConfig) *OutstationLink {
	ctx, cancel := context.WithCancel(context.Background())

	return &OutstationLink{
		localAddress:       config.LocalAddress,
		remoteAddress:      config.RemoteAddress,
		timeout:            config.Timeout,
		state:              LinkStateIdle,
		fcbValidator:       NewFCBValidator(),
		unsolicitedEnabled: false,
		dataCallback:       config.DataCallback,
		statusCallback:     config.StatusCallback,
		sendChan:           make(chan []byte, 10),
		ctx:                ctx,
		cancel:             cancel,
	}
}

// Start starts the link layer
func (o *OutstationLink) Start() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.state != LinkStateIdle {
		return ErrInvalidState
	}

	// Outstation is ready immediately
	return nil
}

// Stop stops the link layer
func (o *OutstationLink) Stop() error {
	o.cancel()
	o.wg.Wait()
	return nil
}

// GetState returns current link state
func (o *OutstationLink) GetState() LinkState {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state
}

// IsOnline returns true if link is operational
func (o *OutstationLink) IsOnline() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state == LinkStateIdle
}

// SetTimeout sets the response timeout
func (o *OutstationLink) SetTimeout(duration time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.timeout = duration
}

// SetRetries sets the maximum number of retries (not used by outstation)
func (o *OutstationLink) SetRetries(count int) {
	// Outstation doesn't retry - this is a no-op
}

// EnableUnsolicited enables unsolicited responses
func (o *OutstationLink) EnableUnsolicited(enable bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.unsolicitedEnabled = enable
}

// OnFrameReceived handles received frames from master
func (o *OutstationLink) OnFrameReceived(frame *Frame) error {
	// Validate frame is from correct source
	if frame.Source != o.remoteAddress || frame.Destination != o.localAddress {
		return ErrInvalidAddress
	}

	// Validate frame is from primary station
	if frame.IsPrimary != PrimaryFrame {
		return fmt.Errorf("outstation received non-primary frame")
	}

	// Validate direction
	if frame.Dir != DirectionMasterToOutstation {
		return ErrInvalidDirection
	}

	o.mu.Lock()
	o.state = LinkStateProcessing
	o.mu.Unlock()

	var err error
	switch frame.FunctionCode {
	case FuncResetLink:
		err = o.handleResetLink(frame)
	case FuncResetUserProcess:
		err = o.handleResetUserProcess(frame)
	case FuncTestLinkStates:
		err = o.handleTestLink(frame)
	case FuncUserDataConfirmed:
		err = o.handleConfirmedUserData(frame)
	case FuncUserDataUnconfirmed:
		err = o.handleUnconfirmedUserData(frame)
	case FuncRequestLinkStatus:
		err = o.handleRequestLinkStatus(frame)
	default:
		err = o.sendLinkNotUsed()
	}

	o.mu.Lock()
	if err == nil {
		o.state = LinkStateIdle
	} else {
		o.state = LinkStateError
	}
	o.mu.Unlock()

	return err
}

// handleResetLink handles RESET LINK command
func (o *OutstationLink) handleResetLink(frame *Frame) error {
	// Reset FCB state
	o.fcbValidator.Reset()

	// Clear any buffers if needed
	// TODO: Clear application layer state if required

	// Send ACK
	return o.sendACK()
}

// handleResetUserProcess handles RESET USER PROCESS command
func (o *OutstationLink) handleResetUserProcess(frame *Frame) error {
	// Clear application layer buffers
	// Note: Does NOT reset FCB state

	// TODO: Signal to application layer to reset

	// Send ACK
	return o.sendACK()
}

// handleTestLink handles TEST LINK STATES command
func (o *OutstationLink) handleTestLink(frame *Frame) error {
	// Simply respond with ACK - link is operational
	return o.sendACK()
}

// handleConfirmedUserData handles confirmed user data
func (o *OutstationLink) handleConfirmedUserData(frame *Frame) error {
	// Check for duplicate using FCB
	isDuplicate := o.fcbValidator.ValidateAndUpdate(frame.FCB, frame.FCV)

	if isDuplicate {
		// Duplicate frame - send ACK but don't reprocess data
		return o.sendACK()
	}

	// New frame - process data
	if o.dataCallback != nil {
		if err := o.dataCallback(frame.UserData); err != nil {
			// Processing error - send NACK
			return o.sendNACK()
		}
	}

	// Send ACK
	return o.sendACK()
}

// handleUnconfirmedUserData handles unconfirmed user data
func (o *OutstationLink) handleUnconfirmedUserData(frame *Frame) error {
	// Process data without sending ACK
	if o.dataCallback != nil {
		return o.dataCallback(frame.UserData)
	}
	return nil
}

// handleRequestLinkStatus handles REQUEST LINK STATUS command
func (o *OutstationLink) handleRequestLinkStatus(frame *Frame) error {
	return o.sendLinkStatus()
}

// SendUnsolicitedData sends unsolicited user data to master
func (o *OutstationLink) SendUnsolicitedData(data []byte) error {
	o.mu.Lock()
	if !o.unsolicitedEnabled {
		o.mu.Unlock()
		return fmt.Errorf("unsolicited responses not enabled")
	}
	o.mu.Unlock()

	frame := NewFrame(
		DirectionOutstationToMaster,
		SecondaryFrame,
		FuncUserDataUnconfirmed,
		o.remoteAddress,
		o.localAddress,
		data,
	)

	return o.transmit(frame)
}

// sendACK sends ACK response
func (o *OutstationLink) sendACK() error {
	frame := NewFrame(
		DirectionOutstationToMaster,
		SecondaryFrame,
		FuncAck,
		o.remoteAddress,
		o.localAddress,
		nil,
	)
	return o.transmit(frame)
}

// sendNACK sends NACK response
func (o *OutstationLink) sendNACK() error {
	frame := NewFrame(
		DirectionOutstationToMaster,
		SecondaryFrame,
		FuncNack,
		o.remoteAddress,
		o.localAddress,
		nil,
	)
	return o.transmit(frame)
}

// sendLinkStatus sends LINK STATUS response
func (o *OutstationLink) sendLinkStatus() error {
	frame := NewFrame(
		DirectionOutstationToMaster,
		SecondaryFrame,
		FuncLinkStatusResponse,
		o.remoteAddress,
		o.localAddress,
		nil,
	)
	return o.transmit(frame)
}

// sendLinkNotUsed sends LINK NOT USED response
func (o *OutstationLink) sendLinkNotUsed() error {
	frame := NewFrame(
		DirectionOutstationToMaster,
		SecondaryFrame,
		FuncLinkNotUsed,
		o.remoteAddress,
		o.localAddress,
		nil,
	)
	return o.transmit(frame)
}

// transmit sends a frame to the physical layer
func (o *OutstationLink) transmit(frame *Frame) error {
	data, err := frame.Serialize()
	if err != nil {
		return err
	}

	select {
	case o.sendChan <- data:
		return nil
	case <-o.ctx.Done():
		return o.ctx.Err()
	}
}

// GetSendChannel returns the channel for sending data to physical layer
func (o *OutstationLink) GetSendChannel() <-chan []byte {
	return o.sendChan
}

// These methods are required by LinkLayer interface but not used by outstation

// ResetLink is not used by outstation (receives reset from master)
func (o *OutstationLink) ResetLink() error {
	return fmt.Errorf("outstation cannot initiate reset link")
}

// ResetUserProcess is not used by outstation
func (o *OutstationLink) ResetUserProcess() error {
	return fmt.Errorf("outstation cannot initiate reset user process")
}

// TestLink is not used by outstation
func (o *OutstationLink) TestLink() error {
	return fmt.Errorf("outstation cannot initiate test link")
}

// RequestLinkStatus is not used by outstation
func (o *OutstationLink) RequestLinkStatus() error {
	return fmt.Errorf("outstation cannot initiate request link status")
}

// SendConfirmedUserData is not used by outstation (uses unsolicited instead)
func (o *OutstationLink) SendConfirmedUserData(data []byte) error {
	return fmt.Errorf("outstation uses unsolicited responses, not confirmed user data")
}

// SendUnconfirmedUserData is used for unsolicited responses
func (o *OutstationLink) SendUnconfirmedUserData(data []byte) error {
	return o.SendUnsolicitedData(data)
}
