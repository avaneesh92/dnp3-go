package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"avaneesh/dnp3-go/pkg/channel"
	"avaneesh/dnp3-go/pkg/dnp3"
	"avaneesh/dnp3-go/pkg/types"
)

// SimpleLogger implements a basic logger for the example
type SimpleLogger struct {
	logger *log.Logger
}

func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *SimpleLogger) Debug(format string, args ...interface{}) {
	l.logger.Printf("[DEBUG] "+format, args...)
}

func (l *SimpleLogger) Info(format string, args ...interface{}) {
	l.logger.Printf("[INFO] "+format, args...)
}

func (l *SimpleLogger) Warn(format string, args ...interface{}) {
	l.logger.Printf("[WARN] "+format, args...)
}

func (l *SimpleLogger) Error(format string, args ...interface{}) {
	l.logger.Printf("[ERROR] "+format, args...)
}

func (l *SimpleLogger) SetLevel(level int) {
	// Simple logger doesn't support level filtering
}

// Example demonstrating TCP channel usage with DNP3

func main() {
	// Create logger
	logger := NewSimpleLogger()

	// Example 1: TCP Client (Master)
	fmt.Println("=== TCP Client Example (Master) ===")
	if err := runTCPMaster(logger); err != nil {
		logger.Error("Master error: %v", err)
	}

	fmt.Println()

	// Example 2: TCP Server (Outstation)
	fmt.Println("=== TCP Server Example (Outstation) ===")
	if err := runTCPOutstation(logger); err != nil {
		logger.Error("Outstation error: %v", err)
	}
}

func runTCPMaster(logger *SimpleLogger) error {
	// Create TCP channel (client mode - connects to remote outstation)
	tcpConfig := channel.TCPChannelConfig{
		Address:        "127.0.0.1:20000", // Connect to outstation
		IsServer:       false,              // Client mode
		ReconnectDelay: 5 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   10 * time.Second,
	}

	tcpChannel, err := channel.NewTCPChannel(tcpConfig)
	if err != nil {
		return fmt.Errorf("failed to create TCP channel: %w", err)
	}
	defer tcpChannel.Close()

	fmt.Printf("TCP Client connected to %s\n", tcpConfig.Address)

	// Create DNP3 channel
	dnp3Channel := channel.New("tcp-master", tcpChannel, logger)
	if err := dnp3Channel.Open(); err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer dnp3Channel.Close()

	// Create DNP3 manager
	manager := dnp3.NewManagerWithLogger(logger)

	// Configure master
	masterConfig := dnp3.DefaultMasterConfig()
	masterConfig.ID = "TCPMaster"
	masterConfig.LocalAddress = 1  // Master address
	masterConfig.RemoteAddress = 10 // Outstation address
	masterConfig.ResponseTimeout = 5 * time.Second

	// Create master callbacks
	callbacks := &MasterCallbacks{logger: logger}

	// Create master
	master, err := manager.CreateMaster(masterConfig, callbacks, dnp3Channel)
	if err != nil {
		return fmt.Errorf("failed to create master: %w", err)
	}

	// Enable master
	if err := master.Enable(); err != nil {
		return fmt.Errorf("failed to enable master: %w", err)
	}

	fmt.Println("Master enabled, performing integrity scan...")

	// Perform integrity scan
	if err := master.ScanIntegrity(); err != nil {
		logger.Error("Integrity scan failed: %v", err)
	}

	// Add periodic integrity scan
	handle, err := master.AddIntegrityScan(60 * time.Second)
	if err != nil {
		logger.Error("Failed to add periodic scan: %v", err)
	} else {
		defer handle.Remove()
		fmt.Println("Added periodic integrity scan (60s)")
	}

	// Run for a bit
	time.Sleep(10 * time.Second)

	// Show statistics
	stats := tcpChannel.Statistics()
	fmt.Printf("\nTCP Statistics:\n")
	fmt.Printf("  Bytes Sent: %d\n", stats.BytesSent)
	fmt.Printf("  Bytes Received: %d\n", stats.BytesReceived)
	fmt.Printf("  Connects: %d\n", stats.Connects)
	fmt.Printf("  Disconnects: %d\n", stats.Disconnects)

	// Shutdown
	if err := master.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown master: %w", err)
	}

	fmt.Println("Master shutdown complete")
	return nil
}

func runTCPOutstation(logger *SimpleLogger) error {
	// Create TCP channel (server mode - listens for incoming connections)
	tcpConfig := channel.TCPChannelConfig{
		Address:      "0.0.0.0:20000", // Listen on all interfaces
		IsServer:     true,             // Server mode
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	tcpChannel, err := channel.NewTCPChannel(tcpConfig)
	if err != nil {
		return fmt.Errorf("failed to create TCP channel: %w", err)
	}
	defer tcpChannel.Close()

	fmt.Printf("TCP Server listening on %s\n", tcpConfig.Address)

	// Create DNP3 channel
	dnp3Channel := channel.New("tcp-outstation", tcpChannel, logger)
	if err := dnp3Channel.Open(); err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer dnp3Channel.Close()

	// Create DNP3 manager
	manager := dnp3.NewManagerWithLogger(logger)

	// Configure outstation
	outstationConfig := dnp3.DefaultOutstationConfig()
	outstationConfig.ID = "TCPOutstation"
	outstationConfig.LocalAddress = 10 // Outstation address
	outstationConfig.RemoteAddress = 1  // Master address

	// Configure database with some points
	outstationConfig.Database = dnp3.DatabaseConfig{
		Binary: []dnp3.BinaryPointConfig{
			{StaticVariation: 1, EventVariation: 2, Class: 1}, // Index 0
			{StaticVariation: 1, EventVariation: 2, Class: 1}, // Index 1
		},
		Analog: []dnp3.AnalogPointConfig{
			{StaticVariation: 30, EventVariation: 32, Class: 2, Deadband: 5.0}, // Index 0
			{StaticVariation: 30, EventVariation: 32, Class: 2, Deadband: 5.0}, // Index 1
		},
	}

	// Create outstation callbacks
	callbacks := &OutstationCallbacks{logger: logger}

	// Create outstation
	outstation, err := manager.CreateOutstation(outstationConfig, callbacks, dnp3Channel)
	if err != nil {
		return fmt.Errorf("failed to create outstation: %w", err)
	}

	// Enable outstation
	if err := outstation.Enable(); err != nil {
		return fmt.Errorf("failed to enable outstation: %w", err)
	}

	fmt.Println("Outstation enabled, updating values...")

	// Update some values
	updates := dnp3.NewUpdateBuilder().
		UpdateBinary(types.Binary{Value: true, Flags: types.FlagOnline}, 0, dnp3.EventModeDetect).
		UpdateBinary(types.Binary{Value: false, Flags: types.FlagOnline}, 1, dnp3.EventModeDetect).
		UpdateAnalog(types.Analog{Value: 123.45, Flags: types.FlagOnline}, 0, dnp3.EventModeDetect).
		UpdateAnalog(types.Analog{Value: 678.90, Flags: types.FlagOnline}, 1, dnp3.EventModeDetect).
		Build()

	if err := outstation.Apply(updates); err != nil {
		logger.Error("Failed to apply updates: %v", err)
	} else {
		fmt.Println("Applied initial values")
	}

	// Run for a bit
	time.Sleep(10 * time.Second)

	// Show statistics
	stats := tcpChannel.Statistics()
	fmt.Printf("\nTCP Statistics:\n")
	fmt.Printf("  Bytes Sent: %d\n", stats.BytesSent)
	fmt.Printf("  Bytes Received: %d\n", stats.BytesReceived)
	fmt.Printf("  Connects: %d\n", stats.Connects)
	fmt.Printf("  Disconnects: %d\n", stats.Disconnects)

	// Shutdown
	if err := outstation.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown outstation: %w", err)
	}

	fmt.Println("Outstation shutdown complete")
	return nil
}

// MasterCallbacks implements dnp3.MasterCallbacks
type MasterCallbacks struct {
	logger *SimpleLogger
}

func (cb *MasterCallbacks) OnBeginFragment(info dnp3.ResponseInfo) {
	cb.logger.Debug("Begin fragment: unsolicited=%v", info.Unsolicited)
}

func (cb *MasterCallbacks) OnEndFragment(info dnp3.ResponseInfo) {
	cb.logger.Debug("End fragment: unsolicited=%v", info.Unsolicited)
}

func (cb *MasterCallbacks) ProcessBinary(info dnp3.HeaderInfo, values []types.IndexedBinary) {
	cb.logger.Info("Received %d binary values", len(values))
	for _, v := range values {
		cb.logger.Info("  Binary[%d]: value=%v, flags=0x%02X", v.Index, v.Value.Value, v.Value.Flags)
	}
}

func (cb *MasterCallbacks) ProcessDoubleBitBinary(info dnp3.HeaderInfo, values []types.IndexedDoubleBitBinary) {
	cb.logger.Info("Received %d double-bit binary values", len(values))
}

func (cb *MasterCallbacks) ProcessAnalog(info dnp3.HeaderInfo, values []types.IndexedAnalog) {
	cb.logger.Info("Received %d analog values", len(values))
	for _, v := range values {
		cb.logger.Info("  Analog[%d]: value=%.2f, flags=0x%02X", v.Index, v.Value.Value, v.Value.Flags)
	}
}

func (cb *MasterCallbacks) ProcessCounter(info dnp3.HeaderInfo, values []types.IndexedCounter) {
	cb.logger.Info("Received %d counter values", len(values))
}

func (cb *MasterCallbacks) ProcessFrozenCounter(info dnp3.HeaderInfo, values []types.IndexedFrozenCounter) {
	cb.logger.Info("Received %d frozen counter values", len(values))
}

func (cb *MasterCallbacks) ProcessBinaryOutputStatus(info dnp3.HeaderInfo, values []types.IndexedBinaryOutputStatus) {
	cb.logger.Info("Received %d binary output status values", len(values))
}

func (cb *MasterCallbacks) ProcessAnalogOutputStatus(info dnp3.HeaderInfo, values []types.IndexedAnalogOutputStatus) {
	cb.logger.Info("Received %d analog output status values", len(values))
}

func (cb *MasterCallbacks) OnReceiveIIN(iin types.IIN) {
	cb.logger.Debug("Received IIN: IIN1=0x%02X, IIN2=0x%02X", iin.IIN1, iin.IIN2)
}

func (cb *MasterCallbacks) OnTaskStart(taskType dnp3.TaskType, id int) {
	cb.logger.Debug("Task started: type=%d, id=%d", taskType, id)
}

func (cb *MasterCallbacks) OnTaskComplete(taskType dnp3.TaskType, id int, result dnp3.TaskResult) {
	cb.logger.Debug("Task complete: type=%d, id=%d, result=%d", taskType, id, result)
}

func (cb *MasterCallbacks) GetTime() time.Time {
	return time.Now()
}

// OutstationCallbacks implements dnp3.OutstationCallbacks
type OutstationCallbacks struct {
	logger *SimpleLogger
}

func (cb *OutstationCallbacks) Begin() {
	cb.logger.Debug("Transaction begin")
}

func (cb *OutstationCallbacks) End() {
	cb.logger.Debug("Transaction end")
}

func (cb *OutstationCallbacks) SelectCROB(crob types.CROB, index uint16) types.CommandStatus {
	cb.logger.Info("SELECT CROB[%d]: opType=%d", index, crob.OpType)
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OperateCROB(crob types.CROB, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	cb.logger.Info("OPERATE CROB[%d]: opType=%d, opMode=%d", index, crob.OpType, opType)
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) SelectAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16) types.CommandStatus {
	cb.logger.Info("SELECT AnalogOutput[%d]: value=%d", index, ao.Value)
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OperateAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	cb.logger.Info("OPERATE AnalogOutput[%d]: value=%d, opType=%d", index, ao.Value, opType)
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) SelectAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OperateAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) SelectAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OperateAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) SelectAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OperateAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OnConfirmReceived(unsolicited bool, numClass1, numClass2, numClass3 uint) {
	cb.logger.Debug("Confirm received: unsolicited=%v", unsolicited)
}

func (cb *OutstationCallbacks) OnUnsolicitedResponse(success bool, seq uint8) {
	cb.logger.Debug("Unsolicited response: success=%v, seq=%d", success, seq)
}

func (cb *OutstationCallbacks) GetApplicationIIN() types.IIN {
	return types.IIN{IIN1: 0, IIN2: 0}
}
