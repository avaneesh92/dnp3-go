package link

import (
	"testing"
	"time"
)

func TestOutstationLink_New(t *testing.T) {
	config := LinkLayerConfig{
		LocalAddress:  1024,
		RemoteAddress: 1,
		IsMaster:      false,
		Timeout:       2 * time.Second,
	}

	outstation := NewOutstationLink(config)

	if outstation.localAddress != 1024 {
		t.Errorf("Expected local address 1024, got %d", outstation.localAddress)
	}

	if outstation.remoteAddress != 1 {
		t.Errorf("Expected remote address 1, got %d", outstation.remoteAddress)
	}

	if outstation.state != LinkStateIdle {
		t.Errorf("Expected state Idle, got %s", outstation.state)
	}

	if outstation.fcbValidator == nil {
		t.Errorf("FCB validator should be initialized")
	}
}

func TestOutstationLink_Start(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)

	err := outstation.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !outstation.IsOnline() {
		t.Errorf("Outstation should be online after start")
	}

	outstation.Stop()
}

func TestOutstationLink_HandleResetLink(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	// Set FCB validator to initialized state
	outstation.fcbValidator.initialized = true
	outstation.fcbValidator.lastFCB = true

	// Send reset link frame
	resetFrame := NewResetLinkFrame(outstation.localAddress, outstation.remoteAddress)

	// Capture sent ACK
	go func() {
		select {
		case data := <-outstation.sendChan:
			// Parse the sent frame
			frame, _, err := Parse(data)
			if err != nil {
				t.Errorf("Failed to parse sent frame: %v", err)
				return
			}
			if frame.FunctionCode != FuncAck {
				t.Errorf("Expected ACK, got function code %d", frame.FunctionCode)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timeout waiting for ACK")
		}
	}()

	err := outstation.OnFrameReceived(resetFrame)
	if err != nil {
		t.Errorf("OnFrameReceived(ResetLink) failed: %v", err)
	}

	// Verify FCB validator was reset
	if outstation.fcbValidator.initialized {
		t.Errorf("FCB validator should be reset")
	}
}

func TestOutstationLink_HandleTestLink(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	testFrame := NewTestLinkFrame(outstation.localAddress, outstation.remoteAddress)

	// Capture sent ACK
	go func() {
		select {
		case data := <-outstation.sendChan:
			frame, _, err := Parse(data)
			if err != nil {
				t.Errorf("Failed to parse sent frame: %v", err)
				return
			}
			if frame.FunctionCode != FuncAck {
				t.Errorf("Expected ACK, got function code %d", frame.FunctionCode)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timeout waiting for ACK")
		}
	}()

	err := outstation.OnFrameReceived(testFrame)
	if err != nil {
		t.Errorf("OnFrameReceived(TestLink) failed: %v", err)
	}
}

func TestOutstationLink_HandleConfirmedUserData_FirstMessage(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false

	dataReceived := false
	config.DataCallback = func(data []byte) error {
		dataReceived = true
		if len(data) != 3 {
			t.Errorf("Expected 3 bytes, got %d", len(data))
		}
		return nil
	}

	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	// First confirmed message with FCB=0
	dataFrame := NewConfirmedUserDataFrame(
		outstation.localAddress,
		outstation.remoteAddress,
		[]byte{0x01, 0x02, 0x03},
		false, // FCB=0
	)

	// Capture sent ACK
	go func() {
		select {
		case data := <-outstation.sendChan:
			frame, _, err := Parse(data)
			if err != nil {
				t.Errorf("Failed to parse sent frame: %v", err)
				return
			}
			if frame.FunctionCode != FuncAck {
				t.Errorf("Expected ACK, got function code %d", frame.FunctionCode)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timeout waiting for ACK")
		}
	}()

	err := outstation.OnFrameReceived(dataFrame)
	if err != nil {
		t.Errorf("OnFrameReceived failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if !dataReceived {
		t.Errorf("Data callback was not called")
	}

	// Verify FCB validator state
	if !outstation.fcbValidator.initialized {
		t.Errorf("FCB validator should be initialized")
	}

	if outstation.fcbValidator.lastFCB != false {
		t.Errorf("Last FCB should be false")
	}
}

func TestOutstationLink_HandleConfirmedUserData_FCBToggle(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false

	receivedData := [][]byte{}
	config.DataCallback = func(data []byte) error {
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		receivedData = append(receivedData, dataCopy)
		return nil
	}

	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	// First message with FCB=0
	frame1 := NewConfirmedUserDataFrame(
		outstation.localAddress,
		outstation.remoteAddress,
		[]byte{0x01},
		false, // FCB=0
	)

	// Consume ACK
	go func() {
		<-outstation.sendChan
	}()

	outstation.OnFrameReceived(frame1)
	time.Sleep(10 * time.Millisecond)

	// Second message with FCB=1 (toggled)
	frame2 := NewConfirmedUserDataFrame(
		outstation.localAddress,
		outstation.remoteAddress,
		[]byte{0x02},
		true, // FCB=1
	)

	go func() {
		<-outstation.sendChan
	}()

	outstation.OnFrameReceived(frame2)
	time.Sleep(10 * time.Millisecond)

	// Should have received both messages
	if len(receivedData) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(receivedData))
	}

	if receivedData[0][0] != 0x01 {
		t.Errorf("First message should be 0x01, got 0x%02X", receivedData[0][0])
	}

	if receivedData[1][0] != 0x02 {
		t.Errorf("Second message should be 0x02, got 0x%02X", receivedData[1][0])
	}
}

func TestOutstationLink_HandleConfirmedUserData_Duplicate(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false

	receivedCount := 0
	config.DataCallback = func(data []byte) error {
		receivedCount++
		return nil
	}

	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	// First message with FCB=0
	frame1 := NewConfirmedUserDataFrame(
		outstation.localAddress,
		outstation.remoteAddress,
		[]byte{0x01, 0x02, 0x03},
		false, // FCB=0
	)

	// Consume ACKs
	go func() {
		for i := 0; i < 2; i++ {
			<-outstation.sendChan
		}
	}()

	outstation.OnFrameReceived(frame1)
	time.Sleep(10 * time.Millisecond)

	// Duplicate message with same FCB=0 (retransmission)
	frame2 := NewConfirmedUserDataFrame(
		outstation.localAddress,
		outstation.remoteAddress,
		[]byte{0x01, 0x02, 0x03},
		false, // FCB=0 (same as before)
	)

	outstation.OnFrameReceived(frame2)
	time.Sleep(10 * time.Millisecond)

	// Should have processed data only once
	if receivedCount != 1 {
		t.Errorf("Expected data processed once, got %d times", receivedCount)
	}
}

func TestOutstationLink_HandleUnconfirmedUserData(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false

	dataReceived := false
	config.DataCallback = func(data []byte) error {
		dataReceived = true
		return nil
	}

	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	unconfFrame := NewUnconfirmedUserDataFrame(
		outstation.localAddress,
		outstation.remoteAddress,
		[]byte{0x01, 0x02, 0x03},
	)

	err := outstation.OnFrameReceived(unconfFrame)
	if err != nil {
		t.Errorf("OnFrameReceived failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if !dataReceived {
		t.Errorf("Data callback was not called")
	}

	// Should NOT send any response
	select {
	case <-outstation.sendChan:
		t.Errorf("Unconfirmed data should not generate response")
	case <-time.After(50 * time.Millisecond):
		// Expected - no response
	}
}

func TestOutstationLink_HandleRequestLinkStatus(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	statusFrame := NewRequestLinkStatusFrame(
		outstation.localAddress,
		outstation.remoteAddress,
	)

	// Capture sent LINK STATUS
	go func() {
		select {
		case data := <-outstation.sendChan:
			frame, _, err := Parse(data)
			if err != nil {
				t.Errorf("Failed to parse sent frame: %v", err)
				return
			}
			if frame.FunctionCode != FuncLinkStatusResponse {
				t.Errorf("Expected LINK STATUS, got function code %d", frame.FunctionCode)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timeout waiting for LINK STATUS")
		}
	}()

	err := outstation.OnFrameReceived(statusFrame)
	if err != nil {
		t.Errorf("OnFrameReceived failed: %v", err)
	}
}

func TestOutstationLink_SendUnsolicited_Disabled(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)

	// Unsolicited disabled by default
	err := outstation.SendUnsolicitedData([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Errorf("Expected error when unsolicited disabled")
	}
}

func TestOutstationLink_SendUnsolicited_Enabled(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)
	outstation.Start()
	defer outstation.Stop()

	// Enable unsolicited
	outstation.EnableUnsolicited(true)

	// Capture sent frame
	go func() {
		select {
		case data := <-outstation.sendChan:
			frame, _, err := Parse(data)
			if err != nil {
				t.Errorf("Failed to parse sent frame: %v", err)
				return
			}
			if frame.FunctionCode != FuncUserDataUnconfirmed {
				t.Errorf("Expected USER DATA UNCONF, got function code %d", frame.FunctionCode)
			}
			if frame.Dir != DirectionOutstationToMaster {
				t.Errorf("Expected direction OutstationToMaster")
			}
			if frame.IsPrimary != SecondaryFrame {
				t.Errorf("Expected secondary frame")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Timeout waiting for unsolicited frame")
		}
	}()

	err := outstation.SendUnsolicitedData([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Errorf("SendUnsolicitedData failed: %v", err)
	}
}

func TestOutstationLink_OnFrameReceived_InvalidAddress(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)

	// Frame from wrong source
	frame := NewTestLinkFrame(outstation.localAddress, 999)

	err := outstation.OnFrameReceived(frame)
	if err != ErrInvalidAddress {
		t.Errorf("Expected ErrInvalidAddress, got %v", err)
	}
}

func TestOutstationLink_OnFrameReceived_InvalidDirection(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)

	// Frame with wrong direction (outstation to master)
	frame := NewFrame(
		DirectionOutstationToMaster, // Wrong direction
		PrimaryFrame,
		FuncTestLinkStates,
		outstation.localAddress,
		outstation.remoteAddress,
		nil,
	)

	err := outstation.OnFrameReceived(frame)
	if err != ErrInvalidDirection {
		t.Errorf("Expected ErrInvalidDirection, got %v", err)
	}
}

func TestFCBValidator_FirstMessage(t *testing.T) {
	validator := NewFCBValidator()

	isDuplicate := validator.ValidateAndUpdate(false, true)

	if isDuplicate {
		t.Errorf("First message should not be duplicate")
	}

	if !validator.initialized {
		t.Errorf("Validator should be initialized after first message")
	}

	if validator.lastFCB != false {
		t.Errorf("Last FCB should be false")
	}
}

func TestFCBValidator_Toggle(t *testing.T) {
	validator := NewFCBValidator()

	// First message FCB=0
	isDuplicate := validator.ValidateAndUpdate(false, true)
	if isDuplicate {
		t.Errorf("First message should not be duplicate")
	}

	// Second message FCB=1 (toggled)
	isDuplicate = validator.ValidateAndUpdate(true, true)
	if isDuplicate {
		t.Errorf("Toggled message should not be duplicate")
	}

	if validator.lastFCB != true {
		t.Errorf("Last FCB should be true")
	}
}

func TestFCBValidator_Duplicate(t *testing.T) {
	validator := NewFCBValidator()

	// First message FCB=0
	validator.ValidateAndUpdate(false, true)

	// Duplicate message FCB=0
	isDuplicate := validator.ValidateAndUpdate(false, true)
	if !isDuplicate {
		t.Errorf("Same FCB should be detected as duplicate")
	}
}

func TestFCBValidator_FCVFalse(t *testing.T) {
	validator := NewFCBValidator()

	// FCV=false means FCB is not valid
	isDuplicate := validator.ValidateAndUpdate(false, false)
	if isDuplicate {
		t.Errorf("FCV=false should never be duplicate")
	}

	if validator.initialized {
		t.Errorf("Validator should not be initialized when FCV=false")
	}
}

func TestFCBValidator_Reset(t *testing.T) {
	validator := NewFCBValidator()

	// Initialize with some state
	validator.ValidateAndUpdate(true, true)

	if !validator.initialized {
		t.Errorf("Should be initialized")
	}

	// Reset
	validator.Reset()

	if validator.initialized {
		t.Errorf("Should not be initialized after reset")
	}

	if validator.lastFCB != false {
		t.Errorf("Last FCB should be false after reset")
	}
}

func TestOutstationLink_MasterOnlyMethods(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.IsMaster = false
	outstation := NewOutstationLink(config)

	// These methods should return errors for outstation
	if err := outstation.ResetLink(); err == nil {
		t.Errorf("ResetLink should fail for outstation")
	}

	if err := outstation.ResetUserProcess(); err == nil {
		t.Errorf("ResetUserProcess should fail for outstation")
	}

	if err := outstation.TestLink(); err == nil {
		t.Errorf("TestLink should fail for outstation")
	}

	if err := outstation.RequestLinkStatus(); err == nil {
		t.Errorf("RequestLinkStatus should fail for outstation")
	}

	if err := outstation.SendConfirmedUserData([]byte{0x01}); err == nil {
		t.Errorf("SendConfirmedUserData should fail for outstation")
	}
}
