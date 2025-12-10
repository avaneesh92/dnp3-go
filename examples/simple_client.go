package main

import (
	"fmt"
	"math/rand"
	"time"

	"avaneesh/dnp3-go/pkg/dnp3"
	"avaneesh/dnp3-go/pkg/types"
	"avaneesh/dnp3-go/pkg/channel"
)

// Example showing how to use the DNP3 library

// Implement Outstation callbacks
type MyOutstationCallbacks struct{}

func (c *MyOutstationCallbacks) Begin() {
	fmt.Println("Command Begin")
}

func (c *MyOutstationCallbacks) End() {
	fmt.Println("Command End")
}

func (c *MyOutstationCallbacks) SelectCROB(crob types.CROB, index uint16) types.CommandStatus {
	fmt.Printf("SELECT CROB[%d]: %v\n", index, crob.OpType)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateCROB(crob types.CROB, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Printf("OPERATE CROB[%d]: %v\n", index, crob.OpType)

	// Update corresponding binary output status
	bos := types.BinaryOutputStatus{
		Value: crob.OpType == types.ControlCodeLatchOn,
		Flags: types.FlagOnline,
		Time:  types.Now(),
	}
	handler.Update(bos, index, dnp3.EventModeForce)

	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OnConfirmReceived(unsolicited bool, numClass1, numClass2, numClass3 uint) {
	fmt.Printf("Confirm received\n")
}

func (c *MyOutstationCallbacks) OnUnsolicitedResponse(success bool, seq uint8) {
	fmt.Printf("Unsolicited response sent: %v\n", success)
}

func (c *MyOutstationCallbacks) GetApplicationIIN() types.IIN {
	return types.IIN{IIN1: 0, IIN2: 0}
}


func exampleOutstation() {
	// Create manager
	manager := dnp3.NewManager()
	// Note: We don't defer Shutdown here because we want to keep running

	// Create your custom transport
	// Create TCP channel (server mode - listens for incoming connections)
	tcpConfig := channel.TCPChannelConfig{
		Address:      "127.0.0.1:22150", // Listen on all interfaces
		IsServer:     true,             // Server mode
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	tcpChannel, err := channel.NewTCPChannel(tcpConfig)
	if err != nil {
		panic(err)
	}

	fmt.Printf("TCP Server listening on %s\n", tcpConfig.Address)


	// Add channel
	channel, err := manager.AddChannel("channel1", tcpChannel)
	if err != nil {
		panic(err)
	}

	// Configure database
	dbConfig := dnp3.DatabaseConfig{
		Binary:  make([]dnp3.BinaryPointConfig, 10),
		Analog:  make([]dnp3.AnalogPointConfig, 10),
		Counter: make([]dnp3.CounterPointConfig, 10),
	}

	// Set point configurations
	for i := range dbConfig.Binary {
		dbConfig.Binary[i] = dnp3.BinaryPointConfig{
			StaticVariation: 1,
			EventVariation:  2,
			Class:           1, // Class 1 events
		}
	}

	for i := range dbConfig.Analog {
		dbConfig.Analog[i] = dnp3.AnalogPointConfig{
			StaticVariation: 1,
			EventVariation:  1,
			Class:           2, // Class 2 events
			Deadband:        0.5,
		}
	}

	// Configure outstation
	config := dnp3.DefaultOutstationConfig()
	config.ID = "outstation1"
	config.LocalAddress = 10
	config.RemoteAddress = 1
	config.Database = dbConfig

	// Create outstation
	outstation, err := channel.AddOutstation(config, &MyOutstationCallbacks{})
	if err != nil {
		panic(err)
	}

	// Enable outstation
	outstation.Enable()

	// Simulate measurement updates with random values
	go func() {
		counter := uint32(0)
		baseTemp := 20.0
		baseVoltage := 120.0

		for {
			time.Sleep(5 * time.Second)

			// Build atomic update
			builder := dnp3.NewUpdateBuilder()

			// Update binary inputs (simulate random breaker states)
			for i := 0; i < 10; i++ {
				builder.UpdateBinary(types.Binary{
					Value: rand.Float64() > 0.3, // 70% chance of being true
					Flags: types.FlagOnline,
					Time:  types.Now(),
				}, uint16(i), dnp3.EventModeDetect)
			}

			// Update analog inputs (simulate random temperature, voltage, etc.)
			for i := 0; i < 10; i++ {
				var value float64
				switch i {
				case 0, 1: // Temperature sensors
					value = baseTemp + (rand.Float64()-0.5)*10 // ±5°C variation
				case 2, 3: // Voltage sensors
					value = baseVoltage + (rand.Float64()-0.5)*20 // ±10V variation
				case 4, 5: // Current sensors
					value = 10.0 + rand.Float64()*50 // 10-60A
				case 6, 7: // Power sensors
					value = 1000.0 + rand.Float64()*5000 // 1-6kW
				default: // Generic values
					value = rand.Float64() * 100
				}

				builder.UpdateAnalog(types.Analog{
					Value: value,
					Flags: types.FlagOnline,
					Time:  types.Now(),
				}, uint16(i), dnp3.EventModeDetect)
			}

			// Update counters (simulate energy meters)
			for i := 0; i < 10; i++ {
				builder.UpdateCounter(types.Counter{
					Value: counter + uint32(i*100),
					Flags: types.FlagOnline,
					Time:  types.Now(),
				}, uint16(i), dnp3.EventModeDetect)
			}

			// Apply updates atomically
			updates := builder.Build()
			if err := outstation.Apply(updates); err != nil {
				fmt.Printf("Update error: %v\n", err)
			} else {
				fmt.Printf("[%s] Updated values - Counter: %d\n", time.Now().Format("15:04:05"), counter)
			}

			counter++
		}
	}()
}

func main() {
	fmt.Println("DNP3-Go Example outstation")

	exampleOutstation()

	// Keep the program running
	fmt.Println("Outstation running. Press Ctrl+C to exit.")
	select {} // Block forever
}
