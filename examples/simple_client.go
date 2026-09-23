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

// State tracks measurement values (similar to C++ opendnp3 example)
type State struct {
	count            uint32
	analogValue      float64
	binaryValue      bool
	doubleBitValue   types.DoubleBitValue
	octetStringValue uint8
}

// AddUpdates processes user input and updates measurements
// Similar to the C++ opendnp3 example AddUpdates function
func AddUpdates(builder *dnp3.UpdateBuilder, state *State, arguments string) {
	for _, c := range arguments {
		switch c {
		case 'c':
			// Increment counter
			builder.UpdateCounter(types.Counter{
				Value: state.count,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)
			state.count++
			fmt.Printf("Counter updated to %d\n", state.count-1)

		case 'a':
			// Increment analog
			builder.UpdateAnalog(types.Analog{
				Value: state.analogValue,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)
			state.analogValue += 1.0
			fmt.Printf("Analog updated to %.2f\n", state.analogValue-1.0)

		case 'b':
			// Toggle binary
			builder.UpdateBinary(types.Binary{
				Value: state.binaryValue,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)
			state.binaryValue = !state.binaryValue
			fmt.Printf("Binary toggled to %v\n", !state.binaryValue)

		case 'd':
			// Toggle double-bit binary
			builder.UpdateDoubleBitBinary(types.DoubleBitBinary{
				Value: state.doubleBitValue,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)
			if state.doubleBitValue == types.DoubleBitOff {
				state.doubleBitValue = types.DoubleBitOn
			} else {
				state.doubleBitValue = types.DoubleBitOff
			}
			fmt.Printf("DoubleBit updated to %v\n", state.doubleBitValue)

		case 'f':
			// Freeze counter (copy counter value to frozen counter)
			builder.UpdateFrozenCounter(types.FrozenCounter{
				Value: state.count,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)
			fmt.Printf("Counter frozen at %d\n", state.count)

		default:
			// Ignore unknown characters
		}
	}
}

func exampleOutstation() {
	// Create manager
	manager := dnp3.NewManager()
	// Note: We don't defer Shutdown here because we want to keep running

	// Create your custom transport
	// Create TCP channel (server mode - listens for incoming connections)
	tcpConfig := channel.TCPChannelConfig{
		Address:      "127.0.0.1:22150", // Listen on all interfaces
		IsServer:     true,              // Server mode
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

	// Configure database (matching C++ opendnp3 example with 10 of each type)
	dbConfig := dnp3.DatabaseConfig{
		Binary:        make([]dnp3.BinaryPointConfig, 1),
		DoubleBit:     make([]dnp3.DoubleBitBinaryPointConfig, 1),
		Analog:        make([]dnp3.AnalogPointConfig, 1),
		Counter:       make([]dnp3.CounterPointConfig, 1),
		FrozenCounter: make([]dnp3.FrozenCounterPointConfig, 1),
		BinaryOutput:  make([]dnp3.BinaryOutputStatusPointConfig, 1),
		AnalogOutput:  make([]dnp3.AnalogOutputStatusPointConfig, 1),
	}

	// Set point configurations
	for i := range dbConfig.Binary {
		dbConfig.Binary[i] = dnp3.BinaryPointConfig{
			StaticVariation: 1,
			EventVariation:  2,
			Class:           1, // Class 1 events
		}
	}

	for i := range dbConfig.DoubleBit {
		dbConfig.DoubleBit[i] = dnp3.DoubleBitBinaryPointConfig{
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

	for i := range dbConfig.Counter {
		dbConfig.Counter[i] = dnp3.CounterPointConfig{
			StaticVariation: 1,
			EventVariation:  1,
			Class:           2,
			Deadband:        0,
		}
	}

	for i := range dbConfig.FrozenCounter {
		dbConfig.FrozenCounter[i] = dnp3.FrozenCounterPointConfig{
			StaticVariation: 1,
			EventVariation:  1,
			Class:           2,
		}
	}

	for i := range dbConfig.BinaryOutput {
		dbConfig.BinaryOutput[i] = dnp3.BinaryOutputStatusPointConfig{
			StaticVariation: 1,
			EventVariation:  2,
			Class:           1,
		}
	}

	for i := range dbConfig.AnalogOutput {
		dbConfig.AnalogOutput[i] = dnp3.AnalogOutputStatusPointConfig{
			StaticVariation: 1,
			EventVariation:  1,
			Class:           2,
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

	// Initialize state
	state := &State{
		count:            0,
		analogValue:      0.0,
		binaryValue:      false,
		doubleBitValue:   types.DoubleBitOff,
		octetStringValue: 1,
	}

	// Channel to signal when to exit
	quit := make(chan bool)

	// Optional: Simulate automatic measurement updates (like the original example)
	// This runs in the background alongside manual updates
	go func() {
		counter := uint32(1000) // Start at 1000 to distinguish from manual updates
		baseTemp := 20.0
		baseVoltage := 120.0

		for {
			select {
			case <-quit:
				return
			case <-time.After(10 * time.Second):
				// Build atomic update
				builder := dnp3.NewUpdateBuilder()

				// Update binary inputs (simulate random breaker states)
				for i := 1; i < 10; i++ { // Start at 1 to leave 0 for manual updates
					builder.UpdateBinary(types.Binary{
						Value: rand.Float64() > 0.3, // 70% chance of being true
						Flags: types.FlagOnline,
						Time:  types.Now(),
					}, uint16(i), dnp3.EventModeDetect)
				}

				// Update analog inputs (simulate random temperature, voltage, etc.)
				for i := 1; i < 10; i++ { // Start at 1 to leave 0 for manual updates
					var value float64
					switch i {
					case 1: // Temperature sensors
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
				for i := 1; i < 10; i++ { // Start at 1 to leave 0 for manual updates
					builder.UpdateCounter(types.Counter{
						Value: counter + uint32(i*100),
						Flags: types.FlagOnline,
						Time:  types.Now(),
					}, uint16(i), dnp3.EventModeDetect)
				}

				// Apply updates atomically
				updates := builder.Build()
				if err := outstation.Apply(updates); err != nil {
					fmt.Printf("Auto-update error: %v\n", err)
				} else {
					fmt.Printf("[%s] Auto-updated values - Counter: %d\n", time.Now().Format("15:04:05"), counter)
				}

				counter++
			}
		}
	}()

	// Interactive loop (matching C++ opendnp3 example)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\nEnter one or more measurement changes then press <enter>")
		fmt.Println("c = counter, b = binary, d = doublebit, a = analog, f = freeze counter, 'quit' = exit")
		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "quit" {
			quit <- true
			fmt.Println("Shutting down outstation...")
			return
		}

		if input == "" {
			continue
		}

		// Update measurements based on input
		builder := dnp3.NewUpdateBuilder()
		AddUpdates(builder, state, input)
		updates := builder.Build()

		if err := outstation.Apply(updates); err != nil {
			fmt.Printf("Update error: %v\n", err)
		} else {
			fmt.Println("Updates applied successfully")
		}
	}
}

func main() {
	fmt.Println("=== DNP3-Go Example Outstation ===")
	fmt.Println("Based on C++ opendnp3 outstation example")
	fmt.Println()

	// Enable debug logging
	dnp3.SetLogLevel(dnp3.LevelDebug)

	// Optional: Enable frame debugging for detailed hex dumps
	dnp3.EnableFrameDebug(false)

	exampleOutstation()

	fmt.Println("Outstation shutdown complete.")
}
