//go:build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/dnp3"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Example showing how to use the DNP3 library

// Step 1: Implement PhysicalChannel for your transport
type SimpleChannel struct {
	// In real implementation, you'd have net.Conn or serial port here
}

func (c *SimpleChannel) Read(ctx context.Context) ([]byte, error) {
	// Read from your transport (TCP, Serial, etc.)
	return nil, nil
}

func (c *SimpleChannel) Write(ctx context.Context, data []byte) error {
	// Write to your transport
	return nil
}

func (c *SimpleChannel) Close() error {
	return nil
}

func (c *SimpleChannel) Statistics() channel.TransportStats {
	return channel.TransportStats{}
}

// Step 2: Implement Master callbacks
type MyMasterCallbacks struct{}

func (c *MyMasterCallbacks) OnBeginFragment(info dnp3.ResponseInfo) {
	fmt.Println("Begin fragment")
}

func (c *MyMasterCallbacks) OnEndFragment(info dnp3.ResponseInfo) {
	fmt.Println("End fragment")
}

func (c *MyMasterCallbacks) ProcessBinary(info dnp3.HeaderInfo, values []types.IndexedBinary) {
	for _, v := range values {
		fmt.Printf("Binary[%d]: %v, Flags: 0x%02X\n", v.Index, v.Value.Value, v.Value.Flags)
	}
}

func (c *MyMasterCallbacks) ProcessDoubleBitBinary(info dnp3.HeaderInfo, values []types.IndexedDoubleBitBinary) {
}
func (c *MyMasterCallbacks) ProcessAnalog(info dnp3.HeaderInfo, values []types.IndexedAnalog) {
	for _, v := range values {
		fmt.Printf("Analog[%d]: %.2f\n", v.Index, v.Value.Value)
	}
}
func (c *MyMasterCallbacks) ProcessCounter(info dnp3.HeaderInfo, values []types.IndexedCounter) {}
func (c *MyMasterCallbacks) ProcessFrozenCounter(info dnp3.HeaderInfo, values []types.IndexedFrozenCounter) {
}
func (c *MyMasterCallbacks) ProcessBinaryOutputStatus(info dnp3.HeaderInfo, values []types.IndexedBinaryOutputStatus) {
}
func (c *MyMasterCallbacks) ProcessAnalogOutputStatus(info dnp3.HeaderInfo, values []types.IndexedAnalogOutputStatus) {
}

func (c *MyMasterCallbacks) OnReceiveIIN(iin types.IIN) {
	fmt.Printf("IIN: [%02X,%02X]\n", iin.IIN1, iin.IIN2)
}

func (c *MyMasterCallbacks) OnTaskStart(taskType dnp3.TaskType, id int) {
	fmt.Printf("Task started: %d\n", taskType)
}

func (c *MyMasterCallbacks) OnTaskComplete(taskType dnp3.TaskType, id int, result dnp3.TaskResult) {
	fmt.Printf("Task complete: %d, result: %d\n", taskType, result)
}

func (c *MyMasterCallbacks) GetTime() time.Time {
	return time.Now()
}

// Step 3: Implement Outstation callbacks
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

// Example usage
func exampleMaster() {
	// Create manager
	manager := dnp3.NewManager()
	defer manager.Shutdown()

	// Create your custom transport
	physicalChannel := &SimpleChannel{}

	// Add channel
	channel, err := manager.AddChannel("channel1", physicalChannel)
	if err != nil {
		panic(err)
	}

	// Configure master
	config := dnp3.DefaultMasterConfig()
	config.ID = "master1"
	config.LocalAddress = 1
	config.RemoteAddress = 10

	// Create master
	master, err := channel.AddMaster(config, &MyMasterCallbacks{})
	if err != nil {
		panic(err)
	}

	// Enable master
	master.Enable()

	// Add periodic integrity scan (every 60 seconds)
	master.AddIntegrityScan(60 * time.Second)

	// Add periodic class 1 scan (every 10 seconds)
	master.AddClassScan(dnp3.Class1, 10*time.Second)

	// Perform one-time integrity scan
	master.ScanIntegrity()

	// Send a control command
	commands := []types.Command{
		{
			Index: 5,
			Type:  types.CommandTypeCROB,
			Data: types.CROB{
				OpType:   types.ControlCodeLatchOn,
				Count:    1,
				OnTimeMs: 1000,
			},
		},
	}

	statuses, err := master.DirectOperate(commands)
	if err != nil {
		fmt.Printf("Command error: %v\n", err)
	} else {
		fmt.Printf("Command statuses: %v\n", statuses)
	}
}

func exampleOutstation() {
	// Create manager
	manager := dnp3.NewManager()
	defer manager.Shutdown()

	// Create your custom transport
	physicalChannel := &SimpleChannel{}

	// Add channel
	channel, err := manager.AddChannel("channel1", physicalChannel)
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

	// Simulate measurement updates
	go func() {
		counter := uint32(0)
		for {
			time.Sleep(5 * time.Second)

			// Build atomic update
			builder := dnp3.NewUpdateBuilder()

			// Update binary
			builder.UpdateBinary(types.Binary{
				Value: counter%2 == 0,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)

			// Update analog
			builder.UpdateAnalog(types.Analog{
				Value: float64(counter) * 1.5,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)

			// Update counter
			builder.UpdateCounter(types.Counter{
				Value: counter,
				Flags: types.FlagOnline,
				Time:  types.Now(),
			}, 0, dnp3.EventModeDetect)

			// Apply updates atomically
			updates := builder.Build()
			if err := outstation.Apply(updates); err != nil {
				fmt.Printf("Update error: %v\n", err)
			}

			counter++
		}
	}()
}

func main() {
	fmt.Println("DNP3-Go Example")
	fmt.Println("This example shows the API usage.")
	fmt.Println("In real use, you'd implement PhysicalChannel for your transport.")

	// Uncomment to run examples:
	// exampleMaster()
	// exampleOutstation()
}
