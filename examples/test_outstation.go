//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/dnp3"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// MyOutstationCallbacks implements the outstation callback interface
type MyOutstationCallbacks struct {
	outstation dnp3.Outstation
}

// Command handler callbacks

func (c *MyOutstationCallbacks) Begin() {
	fmt.Println("[COMMAND] Begin command sequence")
}

func (c *MyOutstationCallbacks) End() {
	fmt.Println("[COMMAND] End command sequence")
}

func (c *MyOutstationCallbacks) SelectCROB(crob types.CROB, index uint16) types.CommandStatus {
	fmt.Printf("[SELECT] CROB[%d]: OpType=%d, Count=%d, OnTime=%dms, OffTime=%dms\n",
		index, crob.OpType, crob.Count, crob.OnTimeMs, crob.OffTimeMs)

	// Validate the command
	if index >= 10 {
		fmt.Printf("[SELECT] CROB[%d]: REJECTED - Invalid index\n", index)
		return types.CommandStatusNotSupported
	}

	fmt.Printf("[SELECT] CROB[%d]: SUCCESS\n", index)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateCROB(crob types.CROB, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Printf("[OPERATE] CROB[%d]: OpType=%d, Count=%d, OnTime=%dms, OffTime=%dms, OperateType=%d\n",
		index, crob.OpType, crob.Count, crob.OnTimeMs, crob.OffTimeMs, opType)

	// Determine new state based on control code
	var newState bool
	switch crob.OpType {
	case types.ControlCodeLatchOn, types.ControlCodePulseOn, types.ControlCodeCloseOn:
		newState = true
	case types.ControlCodeLatchOff, types.ControlCodePulseOff, types.ControlCodeTripOff:
		newState = false
	default:
		fmt.Printf("[OPERATE] CROB[%d]: REJECTED - Unknown OpType\n", index)
		return types.CommandStatusNotSupported
	}

	// Update the corresponding binary output status
	bos := types.BinaryOutputStatus{
		Value: newState,
		Flags: types.FlagOnline,
		Time:  types.Now(),
	}

	// Use the handler to update the point
	handler.Update(bos, index, dnp3.EventModeForce)

	fmt.Printf("[OPERATE] CROB[%d]: SUCCESS - Set to %v\n", index, newState)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16) types.CommandStatus {
	fmt.Printf("[SELECT] AnalogOutputInt32[%d]: Value=%d\n", index, ao.Value)

	if index >= 10 {
		return types.CommandStatusNotSupported
	}

	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Printf("[OPERATE] AnalogOutputInt32[%d]: Value=%d\n", index, ao.Value)

	// Update the corresponding analog output status
	aos := types.AnalogOutputStatus{
		Value: float64(ao.Value),
		Flags: types.FlagOnline,
		Time:  types.Now(),
	}

	handler.Update(aos, index, dnp3.EventModeForce)

	fmt.Printf("[OPERATE] AnalogOutputInt32[%d]: SUCCESS - Set to %d\n", index, ao.Value)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16) types.CommandStatus {
	fmt.Printf("[SELECT] AnalogOutputInt16[%d]: Value=%d\n", index, ao.Value)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Printf("[OPERATE] AnalogOutputInt16[%d]: Value=%d\n", index, ao.Value)

	aos := types.AnalogOutputStatus{
		Value: float64(ao.Value),
		Flags: types.FlagOnline,
		Time:  types.Now(),
	}

	handler.Update(aos, index, dnp3.EventModeForce)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16) types.CommandStatus {
	fmt.Printf("[SELECT] AnalogOutputFloat32[%d]: Value=%.2f\n", index, ao.Value)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Printf("[OPERATE] AnalogOutputFloat32[%d]: Value=%.2f\n", index, ao.Value)

	aos := types.AnalogOutputStatus{
		Value: float64(ao.Value),
		Flags: types.FlagOnline,
		Time:  types.Now(),
	}

	handler.Update(aos, index, dnp3.EventModeForce)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) SelectAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16) types.CommandStatus {
	fmt.Printf("[SELECT] AnalogOutputDouble64[%d]: Value=%.2f\n", index, ao.Value)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OperateAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Printf("[OPERATE] AnalogOutputDouble64[%d]: Value=%.2f\n", index, ao.Value)

	aos := types.AnalogOutputStatus{
		Value: ao.Value,
		Flags: types.FlagOnline,
		Time:  types.Now(),
	}

	handler.Update(aos, index, dnp3.EventModeForce)
	return types.CommandStatusSuccess
}

func (c *MyOutstationCallbacks) OnConfirmReceived(unsolicited bool, numClass1, numClass2, numClass3 uint) {
	fmt.Printf("[CONFIRM] Unsolicited=%v, Class1=%d, Class2=%d, Class3=%d\n",
		unsolicited, numClass1, numClass2, numClass3)
}

func (c *MyOutstationCallbacks) OnUnsolicitedResponse(success bool, seq uint8) {
	fmt.Printf("[UNSOLICITED] Success=%v, Seq=%d\n", success, seq)
}

func (c *MyOutstationCallbacks) GetApplicationIIN() types.IIN {
	// Return default IIN (no flags set)
	return types.IIN{IIN1: 0, IIN2: 0}
}

// Application state for tracking measurements
type OutstationState struct {
	binaryCount        uint32
	analogValue        float64
	counterValue       uint32
	doubleBitState     types.DoubleBitValue
	frozenCounterValue uint32
}

func main() {
	fmt.Println("=== DNP3 Test Outstation ===")
	fmt.Println("Comprehensive outstation with all data types")
	fmt.Println()

	// Enable debug logging
	dnp3.SetLogLevel(dnp3.LevelInfo)

	// Create manager
	manager := dnp3.NewManager()
	defer manager.Shutdown()

	// Create TCP channel (server mode - listens for incoming connections)
	tcpConfig := channel.TCPChannelConfig{
		Address:      "127.0.0.1:20000",
		IsServer:     true,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	tcpChannel, err := channel.NewTCPChannel(tcpConfig)
	if err != nil {
		panic(err)
	}

	fmt.Printf("TCP Server listening on %s\n", tcpConfig.Address)

	// Add channel to manager
	ch, err := manager.AddChannel("channel1", tcpChannel)
	if err != nil {
		panic(err)
	}

	// Configure database with one of each point type
	dbConfig := dnp3.DatabaseConfig{
		Binary:        make([]dnp3.BinaryPointConfig, 10),
		DoubleBit:     make([]dnp3.DoubleBitBinaryPointConfig, 10),
		Analog:        make([]dnp3.AnalogPointConfig, 10),
		Counter:       make([]dnp3.CounterPointConfig, 10),
		FrozenCounter: make([]dnp3.FrozenCounterPointConfig, 10),
		BinaryOutput:  make([]dnp3.BinaryOutputStatusPointConfig, 10),
		AnalogOutput:  make([]dnp3.AnalogOutputStatusPointConfig, 10),
	}

	// Configure binary inputs (variation 2 - with flags)
	for i := range dbConfig.Binary {
		dbConfig.Binary[i] = dnp3.BinaryPointConfig{
			StaticVariation: 2, // Binary with flags
			EventVariation:  2, // Binary event with time
			Class:           1, // Class 1 events
		}
	}

	// Configure double-bit binary inputs
	for i := range dbConfig.DoubleBit {
		dbConfig.DoubleBit[i] = dnp3.DoubleBitBinaryPointConfig{
			StaticVariation: 1, // Double-bit binary
			EventVariation:  2, // Double-bit event with time
			Class:           1,
		}
	}

	// Configure analog inputs (variation 5 - 32-bit float with flags)
	for i := range dbConfig.Analog {
		dbConfig.Analog[i] = dnp3.AnalogPointConfig{
			StaticVariation: 5, // 32-bit float
			EventVariation:  1, // 32-bit with time
			Class:           2, // Class 2 events
			Deadband:        1.0,
		}
	}

	// Configure counters (variation 5 - 32-bit with flags)
	for i := range dbConfig.Counter {
		dbConfig.Counter[i] = dnp3.CounterPointConfig{
			StaticVariation: 5, // 32-bit with flag
			EventVariation:  1, // 32-bit with flag
			Class:           2,
			Deadband:        5,
		}
	}

	// Configure frozen counters
	for i := range dbConfig.FrozenCounter {
		dbConfig.FrozenCounter[i] = dnp3.FrozenCounterPointConfig{
			StaticVariation: 1, // 32-bit with flag
			EventVariation:  1,
			Class:           2,
		}
	}

	// Configure binary output status
	for i := range dbConfig.BinaryOutput {
		dbConfig.BinaryOutput[i] = dnp3.BinaryOutputStatusPointConfig{
			StaticVariation: 1, // Binary output status
			EventVariation:  2, // With time
			Class:           1,
		}
	}

	// Configure analog output status
	for i := range dbConfig.AnalogOutput {
		dbConfig.AnalogOutput[i] = dnp3.AnalogOutputStatusPointConfig{
			StaticVariation: 1, // 32-bit
			EventVariation:  1, // With time
			Class:           2,
			Deadband:        1.0,
		}
	}

	// Create outstation configuration
	config := dnp3.DefaultOutstationConfig()
	config.ID = "test-outstation"
	config.LocalAddress = 10 // Outstation address
	config.RemoteAddress = 1 // Master address
	config.Database = dbConfig
	config.AllowUnsolicited = true
	config.MaxBinaryEvents = 100
	config.MaxAnalogEvents = 100

	// Create callbacks
	callbacks := &MyOutstationCallbacks{}

	// Create outstation
	outstation, err := ch.AddOutstation(config, callbacks)
	if err != nil {
		panic(err)
	}

	callbacks.outstation = outstation

	// Enable outstation
	if err := outstation.Enable(); err != nil {
		panic(err)
	}

	fmt.Println("Outstation enabled and ready")
	fmt.Println()

	// Initialize state
	state := &OutstationState{
		binaryCount:        0,
		analogValue:        20.5,
		counterValue:       0,
		doubleBitState:     types.DoubleBitOff,
		frozenCounterValue: 0,
	}

	// Initialize all points with default values
	builder := dnp3.NewUpdateBuilder()

	// Initialize binary inputs
	for i := 0; i < 10; i++ {
		builder.UpdateBinary(types.Binary{
			Value: false,
			Flags: types.FlagOnline,
			Time:  types.Now(),
		}, uint16(i), dnp3.EventModeSuppress)
	}

	// Initialize analog inputs with different values
	for i := 0; i < 10; i++ {
		builder.UpdateAnalog(types.Analog{
			Value: float64(i) * 10.0,
			Flags: types.FlagOnline,
			Time:  types.Now(),
		}, uint16(i), dnp3.EventModeSuppress)
	}

	// Initialize counters
	for i := 0; i < 10; i++ {
		builder.UpdateCounter(types.Counter{
			Value: uint32(i * 100),
			Flags: types.FlagOnline,
			Time:  types.Now(),
		}, uint16(i), dnp3.EventModeSuppress)
	}

	// Initialize double-bit binaries
	for i := 0; i < 10; i++ {
		builder.UpdateDoubleBitBinary(types.DoubleBitBinary{
			Value: types.DoubleBitOff,
			Flags: types.FlagOnline,
			Time:  types.Now(),
		}, uint16(i), dnp3.EventModeSuppress)
	}

	// Initialize frozen counters
	for i := 0; i < 10; i++ {
		builder.UpdateFrozenCounter(types.FrozenCounter{
			Value: 0,
			Flags: types.FlagOnline,
			Time:  types.Now(),
		}, uint16(i), dnp3.EventModeSuppress)
	}

	// Initialize binary output status
	for i := 0; i < 10; i++ {
		builder.UpdateBinaryOutputStatus(types.BinaryOutputStatus{
			Value: false,
			Flags: types.FlagOnline,
			Time:  types.Now(),
		}, uint16(i), dnp3.EventModeSuppress)
	}

	// Initialize analog output status
	for i := 0; i < 10; i++ {
		builder.UpdateAnalogOutputStatus(types.AnalogOutputStatus{
			Value: 0.0,
			Flags: types.FlagOnline,
			Time:  types.Now(),
		}, uint16(i), dnp3.EventModeSuppress)
	}

	// Apply initial values
	if err := outstation.Apply(builder.Build()); err != nil {
		fmt.Printf("Error applying initial values: %v\n", err)
	} else {
		fmt.Println("Initial values applied")
	}

	// Print configured points summary
	printPointsSummary(dbConfig)

	// Channel to signal exit
	quit := make(chan bool)

	// Background goroutine for automatic updates
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				// Simulate automatic measurement updates
				builder := dnp3.NewUpdateBuilder()

				// Update some binary inputs randomly
				for i := 1; i < 5; i++ {
					builder.UpdateBinary(types.Binary{
						Value: rand.Float64() > 0.5,
						Flags: types.FlagOnline,
						Time:  types.Now(),
					}, uint16(i), dnp3.EventModeDetect)
				}

				// Update analog inputs with simulated sensor readings
				temperature := 20.0 + (rand.Float64()-0.5)*10.0
				voltage := 120.0 + (rand.Float64()-0.5)*5.0
				current := 10.0 + rand.Float64()*20.0

				builder.UpdateAnalog(types.Analog{
					Value: temperature,
					Flags: types.FlagOnline,
					Time:  types.Now(),
				}, 1, dnp3.EventModeDetect)

				builder.UpdateAnalog(types.Analog{
					Value: voltage,
					Flags: types.FlagOnline,
					Time:  types.Now(),
				}, 2, dnp3.EventModeDetect)

				builder.UpdateAnalog(types.Analog{
					Value: current,
					Flags: types.FlagOnline,
					Time:  types.Now(),
				}, 3, dnp3.EventModeDetect)

				// Increment counter
				state.counterValue++
				builder.UpdateCounter(types.Counter{
					Value: state.counterValue,
					Flags: types.FlagOnline,
					Time:  types.Now(),
				}, 1, dnp3.EventModeDetect)

				// Apply updates
				if err := outstation.Apply(builder.Build()); err != nil {
					fmt.Printf("Error applying auto-updates: %v\n", err)
				} else {
					fmt.Printf("[AUTO-UPDATE] Temperature=%.1f°C, Voltage=%.1fV, Current=%.1fA, Counter=%d\n",
						temperature, voltage, current, state.counterValue)
				}
			}
		}
	}()

	// Interactive command loop
	scanner := bufio.NewScanner(os.Stdin)
	printHelp()

	for {
		fmt.Print("\n> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		cmd := parts[0]

		switch cmd {
		case "help", "h", "?":
			printHelp()

		case "quit", "exit", "q":
			quit <- true
			fmt.Println("Shutting down outstation...")
			outstation.Shutdown()
			return

		case "b", "binary":
			// Toggle binary input
			index := uint16(0)
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%d", &index)
			}

			state.binaryCount++
			builder := dnp3.NewUpdateBuilder()
			builder.UpdateBinary(types.Binary{
				Value: state.binaryCount%2 == 1,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, index, dnp3.EventModeForce)

			if err := outstation.Apply(builder.Build()); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Binary[%d] toggled to %v\n", index, state.binaryCount%2 == 1)
			}

		case "a", "analog":
			// Update analog input
			index := uint16(0)
			value := state.analogValue
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%d", &index)
			}
			if len(parts) > 2 {
				fmt.Sscanf(parts[2], "%f", &value)
			} else {
				state.analogValue += 1.0
				value = state.analogValue
			}

			builder := dnp3.NewUpdateBuilder()
			builder.UpdateAnalog(types.Analog{
				Value: value,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, index, dnp3.EventModeForce)

			if err := outstation.Apply(builder.Build()); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Analog[%d] = %.2f\n", index, value)
			}

		case "c", "counter":
			// Increment counter
			index := uint16(0)
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%d", &index)
			}

			state.counterValue++
			builder := dnp3.NewUpdateBuilder()
			builder.UpdateCounter(types.Counter{
				Value: state.counterValue,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, index, dnp3.EventModeForce)

			if err := outstation.Apply(builder.Build()); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Counter[%d] = %d\n", index, state.counterValue)
			}

		case "d", "doublebit":
			// Toggle double-bit binary
			index := uint16(0)
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%d", &index)
			}

			if state.doubleBitState == types.DoubleBitOff {
				state.doubleBitState = types.DoubleBitOn
			} else {
				state.doubleBitState = types.DoubleBitOff
			}

			builder := dnp3.NewUpdateBuilder()
			builder.UpdateDoubleBitBinary(types.DoubleBitBinary{
				Value: state.doubleBitState,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, index, dnp3.EventModeForce)

			if err := outstation.Apply(builder.Build()); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("DoubleBit[%d] = %v\n", index, state.doubleBitState)
			}

		case "f", "freeze":
			// Freeze counter
			index := uint16(0)
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%d", &index)
			}

			state.frozenCounterValue = state.counterValue
			builder := dnp3.NewUpdateBuilder()
			builder.UpdateFrozenCounter(types.FrozenCounter{
				Value: state.frozenCounterValue,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, index, dnp3.EventModeForce)

			if err := outstation.Apply(builder.Build()); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("FrozenCounter[%d] = %d\n", index, state.frozenCounterValue)
			}

		case "status":
			fmt.Println("\n=== Outstation Status ===")
			fmt.Printf("Binary count: %d\n", state.binaryCount)
			fmt.Printf("Analog value: %.2f\n", state.analogValue)
			fmt.Printf("Counter: %d\n", state.counterValue)
			fmt.Printf("DoubleBit: %v\n", state.doubleBitState)
			fmt.Printf("Frozen counter: %d\n", state.frozenCounterValue)

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Type 'help' for available commands")
		}
	}
}

func printPointsSummary(config dnp3.DatabaseConfig) {
	fmt.Println("\n=== Configured Points ===")

	// Binary Inputs
	if len(config.Binary) > 0 {
		fmt.Printf("\nBinary Inputs (Group 1): %d points\n", len(config.Binary))
		fmt.Printf("  Static Variation: %d, Event Variation: %d, Class: %d\n",
			config.Binary[0].StaticVariation,
			config.Binary[0].EventVariation,
			config.Binary[0].Class)
		fmt.Printf("  Indices: 0-%d\n", len(config.Binary)-1)
		fmt.Println("  Description: Digital inputs (breaker states, alarm contacts, etc.)")
	}

	// Double-bit Binary Inputs
	if len(config.DoubleBit) > 0 {
		fmt.Printf("\nDouble-bit Binary Inputs (Group 3): %d points\n", len(config.DoubleBit))
		fmt.Printf("  Static Variation: %d, Event Variation: %d, Class: %d\n",
			config.DoubleBit[0].StaticVariation,
			config.DoubleBit[0].EventVariation,
			config.DoubleBit[0].Class)
		fmt.Printf("  Indices: 0-%d\n", len(config.DoubleBit)-1)
		fmt.Println("  Description: 4-state inputs (OFF, ON, INTERMEDIATE, INDETERMINATE)")
	}

	// Analog Inputs
	if len(config.Analog) > 0 {
		fmt.Printf("\nAnalog Inputs (Group 30): %d points\n", len(config.Analog))
		fmt.Printf("  Static Variation: %d (32-bit float), Event Variation: %d, Class: %d\n",
			config.Analog[0].StaticVariation,
			config.Analog[0].EventVariation,
			config.Analog[0].Class)
		fmt.Printf("  Deadband: %.1f\n", config.Analog[0].Deadband)
		fmt.Printf("  Indices: 0-%d\n", len(config.Analog)-1)
		fmt.Println("  Description: Analog measurements (temperature, voltage, current, power, etc.)")
		fmt.Println("  Auto-update points: 1=Temperature, 2=Voltage, 3=Current")
	}

	// Counters
	if len(config.Counter) > 0 {
		fmt.Printf("\nCounters (Group 20): %d points\n", len(config.Counter))
		fmt.Printf("  Static Variation: %d (32-bit with flag), Event Variation: %d, Class: %d\n",
			config.Counter[0].StaticVariation,
			config.Counter[0].EventVariation,
			config.Counter[0].Class)
		fmt.Printf("  Deadband: %d\n", config.Counter[0].Deadband)
		fmt.Printf("  Indices: 0-%d\n", len(config.Counter)-1)
		fmt.Println("  Description: Accumulator values (energy meters, pulse counts, etc.)")
		fmt.Println("  Auto-update point: 1=Energy counter")
	}

	// Frozen Counters
	if len(config.FrozenCounter) > 0 {
		fmt.Printf("\nFrozen Counters (Group 21): %d points\n", len(config.FrozenCounter))
		fmt.Printf("  Static Variation: %d, Event Variation: %d, Class: %d\n",
			config.FrozenCounter[0].StaticVariation,
			config.FrozenCounter[0].EventVariation,
			config.FrozenCounter[0].Class)
		fmt.Printf("  Indices: 0-%d\n", len(config.FrozenCounter)-1)
		fmt.Println("  Description: Snapshot of counter values at specific time")
	}

	// Binary Output Status
	if len(config.BinaryOutput) > 0 {
		fmt.Printf("\nBinary Output Status (Group 10): %d points\n", len(config.BinaryOutput))
		fmt.Printf("  Static Variation: %d, Event Variation: %d, Class: %d\n",
			config.BinaryOutput[0].StaticVariation,
			config.BinaryOutput[0].EventVariation,
			config.BinaryOutput[0].Class)
		fmt.Printf("  Indices: 0-%d\n", len(config.BinaryOutput)-1)
		fmt.Println("  Description: Output point status (updated by CROB commands)")
	}

	// Analog Output Status
	if len(config.AnalogOutput) > 0 {
		fmt.Printf("\nAnalog Output Status (Group 40): %d points\n", len(config.AnalogOutput))
		fmt.Printf("  Static Variation: %d, Event Variation: %d, Class: %d\n",
			config.AnalogOutput[0].StaticVariation,
			config.AnalogOutput[0].EventVariation,
			config.AnalogOutput[0].Class)
		fmt.Printf("  Deadband: %.1f\n", config.AnalogOutput[0].Deadband)
		fmt.Printf("  Indices: 0-%d\n", len(config.AnalogOutput)-1)
		fmt.Println("  Description: Analog output point status (updated by Analog Output commands)")
	}

	fmt.Println("\n=== Supported Commands ===")
	fmt.Println("  CROB (Group 12): Control Relay Output Block")
	fmt.Println("    - Latch On/Off, Pulse On/Off, Close, Trip")
	fmt.Println("    - Updates Binary Output Status points")
	fmt.Println("  Analog Output (Group 41): Analog output commands")
	fmt.Println("    - Int32, Int16, Float32, Double64 variations")
	fmt.Println("    - Updates Analog Output Status points")

	fmt.Println("\n=== Event Classes ===")
	fmt.Println("  Class 1: Binary inputs, Double-bit, Binary output status")
	fmt.Println("  Class 2: Analog inputs, Counters, Frozen counters, Analog output status")
	fmt.Println("  Class 3: (none configured)")
	fmt.Println()
}

func printHelp() {
	fmt.Println("\n=== Available Commands ===")
	fmt.Println("  b [index]         - Toggle binary input (default index 0)")
	fmt.Println("  a [index] [value] - Update analog input (default index 0, auto-increment)")
	fmt.Println("  c [index]         - Increment counter (default index 0)")
	fmt.Println("  d [index]         - Toggle double-bit binary (default index 0)")
	fmt.Println("  f [index]         - Freeze counter value (default index 0)")
	fmt.Println("  status            - Show current state")
	fmt.Println("  help              - Show this help")
	fmt.Println("  quit              - Exit program")
	fmt.Println("\nNote: Auto-updates run every 5 seconds in the background")
}
