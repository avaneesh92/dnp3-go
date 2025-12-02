package app

// FunctionCode represents DNP3 application function codes
type FunctionCode uint8

// Application function codes
const (
	FuncConfirm               FunctionCode = 0x00 // Confirm
	FuncRead                  FunctionCode = 0x01 // Read
	FuncWrite                 FunctionCode = 0x02 // Write
	FuncSelect                FunctionCode = 0x03 // Select
	FuncOperate               FunctionCode = 0x04 // Operate
	FuncDirectOperate         FunctionCode = 0x05 // Direct Operate
	FuncDirectOperateNoAck    FunctionCode = 0x06 // Direct Operate No Ack
	FuncImmediateFreeze       FunctionCode = 0x07 // Immediate Freeze
	FuncImmediateFreezeNoAck  FunctionCode = 0x08 // Immediate Freeze No Ack
	FuncFreezeClear           FunctionCode = 0x09 // Freeze Clear
	FuncFreezeClearNoAck      FunctionCode = 0x0A // Freeze Clear No Ack
	FuncFreezeAtTime          FunctionCode = 0x0B // Freeze At Time
	FuncFreezeAtTimeNoAck     FunctionCode = 0x0C // Freeze At Time No Ack
	FuncColdRestart           FunctionCode = 0x0D // Cold Restart
	FuncWarmRestart           FunctionCode = 0x0E // Warm Restart
	FuncInitializeData        FunctionCode = 0x0F // Initialize Data
	FuncInitializeApplication FunctionCode = 0x10 // Initialize Application
	FuncStartApplication      FunctionCode = 0x11 // Start Application
	FuncStopApplication       FunctionCode = 0x12 // Stop Application
	FuncSaveCon figuration      FunctionCode = 0x13 // Save Configuration
	FuncEnableUnsolicited     FunctionCode = 0x14 // Enable Unsolicited
	FuncDisableUnsolicited    FunctionCode = 0x15 // Disable Unsolicited
	FuncAssignClass           FunctionCode = 0x16 // Assign Class
	FuncDelayMeasurement      FunctionCode = 0x17 // Delay Measurement
	FuncRecordCurrentTime     FunctionCode = 0x18 // Record Current Time
	FuncOpenFile              FunctionCode = 0x19 // Open File
	FuncCloseFile             FunctionCode = 0x1A // Close File
	FuncDeleteFile            FunctionCode = 0x1B // Delete File
	FuncGetFileInfo           FunctionCode = 0x1C // Get File Info
	FuncAuthenticateFile      FunctionCode = 0x1D // Authenticate File
	FuncAbortFile             FunctionCode = 0x1E // Abort File
	FuncResponse              FunctionCode = 0x81 // Response
	FuncUnsolicitedResponse   FunctionCode = 0x82 // Unsolicited Response
	FuncAuthResponse          FunctionCode = 0x83 // Auth Response
)

// String returns string representation of function code
func (f FunctionCode) String() string {
	switch f {
	case FuncConfirm:
		return "Confirm"
	case FuncRead:
		return "Read"
	case FuncWrite:
		return "Write"
	case FuncSelect:
		return "Select"
	case FuncOperate:
		return "Operate"
	case FuncDirectOperate:
		return "DirectOperate"
	case FuncDirectOperateNoAck:
		return "DirectOperateNoAck"
	case FuncResponse:
		return "Response"
	case FuncUnsolicitedResponse:
		return "UnsolicitedResponse"
	case FuncEnableUnsolicited:
		return "EnableUnsolicited"
	case FuncDisableUnsolicited:
		return "DisableUnsolicited"
	default:
		return "Unknown"
	}
}

// IsRequest returns true if this is a request function code
func (f FunctionCode) IsRequest() bool {
	return f != FuncResponse && f != FuncUnsolicitedResponse && f != FuncAuthResponse
}

// IsResponse returns true if this is a response function code
func (f FunctionCode) IsResponse() bool {
	return f == FuncResponse || f == FuncUnsolicitedResponse || f == FuncAuthResponse
}
