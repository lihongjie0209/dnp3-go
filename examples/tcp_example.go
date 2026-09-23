//go:build ignore

package main

import (
	"fmt"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/dnp3"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Example demonstrating TCP channel usage with DNP3

func main() {

	// Example 1: TCP Client (Master)
	fmt.Println("=== TCP Client Example (Master) ===")
	if err := runTCPMaster(); err != nil {
		fmt.Println("Master error: %v", err)
	}

	fmt.Println()

	// Example 2: TCP Server (Outstation)
	fmt.Println("=== TCP Server Example (Outstation) ===")
	if err := runTCPOutstation(); err != nil {
		fmt.Println("Outstation error: %v", err)
	}
}

func runTCPMaster() error {
	// Create TCP channel (client mode - connects to remote outstation)
	tcpConfig := channel.TCPChannelConfig{
		Address:        "127.0.0.1:20000", // Connect to outstation
		IsServer:       false,             // Client mode
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

	// Create DNP3 manager
	// Create manager
	manager := dnp3.NewManager()
	defer manager.Shutdown()

	// Add channel
	manager.AddChannel("channel1", tcpChannel)

	// Configure master
	masterConfig := dnp3.DefaultMasterConfig()
	masterConfig.ID = "TCPMaster"
	masterConfig.LocalAddress = 1   // Master address
	masterConfig.RemoteAddress = 10 // Outstation address
	masterConfig.ResponseTimeout = 5 * time.Second

	// Create master callbacks
	callbacks := &MasterCallbacks{}

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
		fmt.Println("Integrity scan failed: %v", err)
	}

	// Add periodic integrity scan
	handle, err := master.AddIntegrityScan(60 * time.Second)
	if err != nil {
		fmt.Println("Failed to add periodic scan: %v", err)
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

func runTCPOutstation() error {
	// Create TCP channel (server mode - listens for incoming connections)
	tcpConfig := channel.TCPChannelConfig{
		Address:      "0.0.0.0:20000", // Listen on all interfaces
		IsServer:     true,            // Server mode
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	tcpChannel, err := channel.NewTCPChannel(tcpConfig)
	if err != nil {
		return fmt.Errorf("failed to create TCP channel: %w", err)
	}
	defer tcpChannel.Close()

	fmt.Printf("TCP Server listening on %s\n", tcpConfig.Address)

	// Create DNP3 manager
	manager := dnp3.NewManagerWith()
	defer manager.Shutdown()
	// Add channel
	channel, err := manager.AddChannel("channel1", physicalChannel)
	if err != nil {
		panic(err)
	}

	// Configure outstation
	outstationConfig := dnp3.DefaultOutstationConfig()
	outstationConfig.ID = "TCPOutstation"
	outstationConfig.LocalAddress = 10 // Outstation address
	outstationConfig.RemoteAddress = 1 // Master address

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
	callbacks := &OutstationCallbacks{}

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
		fmt.Println("Failed to apply updates: %v", err)
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
}

func (cb *MasterCallbacks) OnBeginFragment(info dnp3.ResponseInfo) {
	fmt.Println("Begin fragment: unsolicited=%v", info.Unsolicited)
}

func (cb *MasterCallbacks) OnEndFragment(info dnp3.ResponseInfo) {
	fmt.Println("End fragment: unsolicited=%v", info.Unsolicited)
}

func (cb *MasterCallbacks) ProcessBinary(info dnp3.HeaderInfo, values []types.IndexedBinary) {
	fmt.Println("Received %d binary values", len(values))
	for _, v := range values {
		fmt.Println("  Binary[%d]: value=%v, flags=0x%02X", v.Index, v.Value.Value, v.Value.Flags)
	}
}

func (cb *MasterCallbacks) ProcessDoubleBitBinary(info dnp3.HeaderInfo, values []types.IndexedDoubleBitBinary) {
	fmt.Println("Received %d double-bit binary values", len(values))
}

func (cb *MasterCallbacks) ProcessAnalog(info dnp3.HeaderInfo, values []types.IndexedAnalog) {
	fmt.Println("Received %d analog values", len(values))
	for _, v := range values {
		fmt.Println("  Analog[%d]: value=%.2f, flags=0x%02X", v.Index, v.Value.Value, v.Value.Flags)
	}
}

func (cb *MasterCallbacks) ProcessCounter(info dnp3.HeaderInfo, values []types.IndexedCounter) {
	fmt.Println("Received %d counter values", len(values))
}

func (cb *MasterCallbacks) ProcessFrozenCounter(info dnp3.HeaderInfo, values []types.IndexedFrozenCounter) {
	fmt.Println("Received %d frozen counter values", len(values))
}

func (cb *MasterCallbacks) ProcessBinaryOutputStatus(info dnp3.HeaderInfo, values []types.IndexedBinaryOutputStatus) {
	fmt.Println("Received %d binary output status values", len(values))
}

func (cb *MasterCallbacks) ProcessAnalogOutputStatus(info dnp3.HeaderInfo, values []types.IndexedAnalogOutputStatus) {
	fmt.Println("Received %d analog output status values", len(values))
}

func (cb *MasterCallbacks) OnReceiveIIN(iin types.IIN) {
	fmt.Println("Received IIN: IIN1=0x%02X, IIN2=0x%02X", iin.IIN1, iin.IIN2)
}

func (cb *MasterCallbacks) OnTaskStart(taskType dnp3.TaskType, id int) {
	fmt.Println("Task started: type=%d, id=%d", taskType, id)
}

func (cb *MasterCallbacks) OnTaskComplete(taskType dnp3.TaskType, id int, result dnp3.TaskResult) {
	fmt.Println("Task complete: type=%d, id=%d, result=%d", taskType, id, result)
}

func (cb *MasterCallbacks) GetTime() time.Time {
	return time.Now()
}

// OutstationCallbacks implements dnp3.OutstationCallbacks
type OutstationCallbacks struct {
}

func (cb *OutstationCallbacks) Begin() {
	fmt.Println("Transaction begin")
}

func (cb *OutstationCallbacks) End() {
	fmt.Println("Transaction end")
}

func (cb *OutstationCallbacks) SelectCROB(crob types.CROB, index uint16) types.CommandStatus {
	fmt.Println("SELECT CROB[%d]: opType=%d", index, crob.OpType)
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OperateCROB(crob types.CROB, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Println("OPERATE CROB[%d]: opType=%d, opMode=%d", index, crob.OpType, opType)
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) SelectAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16) types.CommandStatus {
	fmt.Println("SELECT AnalogOutput[%d]: value=%d", index, ao.Value)
	return types.CommandStatusSuccess
}

func (cb *OutstationCallbacks) OperateAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16, opType dnp3.OperateType, handler dnp3.UpdateHandler) types.CommandStatus {
	fmt.Println("OPERATE AnalogOutput[%d]: value=%d, opType=%d", index, ao.Value, opType)
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
	fmt.Println("Confirm received: unsolicited=%v", unsolicited)
}

func (cb *OutstationCallbacks) OnUnsolicitedResponse(success bool, seq uint8) {
	fmt.Println("Unsolicited response: success=%v, seq=%d", success, seq)
}

func (cb *OutstationCallbacks) GetApplicationIIN() types.IIN {
	return types.IIN{IIN1: 0, IIN2: 0}
}
