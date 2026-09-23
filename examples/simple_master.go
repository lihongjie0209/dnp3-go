//go:build ignore

package main

import (
	"fmt"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/dnp3"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Master callbacks implementation
type MyMasterCallbacks struct {
	lastPrintTime time.Time
}

func (c *MyMasterCallbacks) OnBeginFragment(info dnp3.ResponseInfo) {
	// Print for unsolicited responses to show they're being received
	if info.Unsolicited {
		fmt.Printf("\n>>> Unsolicited Response Received <<<\n")
	}
}

func (c *MyMasterCallbacks) OnEndFragment(info dnp3.ResponseInfo) {
	// Print end marker for unsolicited responses
	if info.Unsolicited {
		fmt.Printf(">>> End Unsolicited Response <<<\n")
	}
}

func (c *MyMasterCallbacks) ProcessBinary(info dnp3.HeaderInfo, values []types.IndexedBinary) {
	if len(values) == 0 {
		return
	}

	fmt.Printf("\n=== Binary Inputs (Breaker States) ===\n")
	for _, v := range values {
		status := "OFF"
		if v.Value.Value {
			status = "ON "
		}
		fmt.Printf("  Breaker[%d]: %s  (flags=0x%02X)\n", v.Index, status, v.Value.Flags)
	}
}

func (c *MyMasterCallbacks) ProcessDoubleBitBinary(info dnp3.HeaderInfo, values []types.IndexedDoubleBitBinary) {
	// Not used in this example
}

func (c *MyMasterCallbacks) ProcessAnalog(info dnp3.HeaderInfo, values []types.IndexedAnalog) {
	if len(values) == 0 {
		return
	}

	fmt.Printf("\n=== Analog Inputs (Sensors) ===\n")
	for _, v := range values {
		var description string
		var unit string

		switch v.Index {
		case 0, 1:
			description = "Temperature"
			unit = "°C"
		case 2, 3:
			description = "Voltage    "
			unit = "V"
		case 4, 5:
			description = "Current    "
			unit = "A"
		case 6, 7:
			description = "Power      "
			unit = "W"
		default:
			description = "Generic    "
			unit = ""
		}

		fmt.Printf("  %s[%d]: %8.2f %s  (flags=0x%02X)\n",
			description, v.Index, v.Value.Value, unit, v.Value.Flags)
	}
}

func (c *MyMasterCallbacks) ProcessCounter(info dnp3.HeaderInfo, values []types.IndexedCounter) {
	if len(values) == 0 {
		return
	}

	fmt.Printf("\n=== Counters (Energy Meters) ===\n")
	for _, v := range values {
		fmt.Printf("  Energy Meter[%d]: %10d kWh  (flags=0x%02X)\n",
			v.Index, v.Value.Value, v.Value.Flags)
	}

	// Print timestamp after all data
	fmt.Printf("\n[%s] Data received from outstation\n", time.Now().Format("15:04:05"))
	fmt.Println("=====================================")
}

func (c *MyMasterCallbacks) ProcessFrozenCounter(info dnp3.HeaderInfo, values []types.IndexedFrozenCounter) {
	fmt.Printf("Received %d frozen counter values\n", len(values))
}

func (c *MyMasterCallbacks) ProcessBinaryOutputStatus(info dnp3.HeaderInfo, values []types.IndexedBinaryOutputStatus) {
	fmt.Printf("Received %d binary output status values\n", len(values))
}

func (c *MyMasterCallbacks) ProcessAnalogOutputStatus(info dnp3.HeaderInfo, values []types.IndexedAnalogOutputStatus) {
	fmt.Printf("Received %d analog output status values\n", len(values))
}

func (c *MyMasterCallbacks) OnReceiveIIN(iin types.IIN) {
	// Only print if there are error flags set
	if iin.IIN1 != 0 || iin.IIN2 != 0 {
		fmt.Printf("\n[WARNING] IIN Flags: IIN1=0x%02X, IIN2=0x%02X\n", iin.IIN1, iin.IIN2)
	}
}

func (c *MyMasterCallbacks) OnTaskStart(taskType dnp3.TaskType, id int) {
	// Don't print - too verbose
}

func (c *MyMasterCallbacks) OnTaskComplete(taskType dnp3.TaskType, id int, result dnp3.TaskResult) {
	// Only print failures
	if result != 0 { // Assuming 0 is success
		fmt.Printf("\n[ERROR] Task failed: type=%d, id=%d, result=%d\n", taskType, id, result)
	}
}

func (c *MyMasterCallbacks) GetTime() time.Time {
	return time.Now()
}

func main() {
	fmt.Println("DNP3-Go Example Master")
	fmt.Println("Connecting to outstation at 127.0.0.1:20000")

	// Create TCP channel (client mode - connects to outstation)
	tcpConfig := channel.TCPChannelConfig{
		Address:        "127.0.0.1:20000", // Connect to outstation
		IsServer:       false,             // Client mode
		ReconnectDelay: 5 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   10 * time.Second,
	}

	tcpChannel, err := channel.NewTCPChannel(tcpConfig)
	if err != nil {
		fmt.Printf("Failed to create TCP channel: %v\n", err)
		return
	}

	fmt.Printf("TCP Client connected to %s\n", tcpConfig.Address)

	// Create DNP3 manager
	manager := dnp3.NewManager()
	// Note: We don't defer Shutdown here because we want to keep running

	// Add channel
	dnp3Channel, err := manager.AddChannel("channel1", tcpChannel)
	if err != nil {
		fmt.Printf("Failed to add channel: %v\n", err)
		return
	}

	// Configure master
	masterConfig := dnp3.DefaultMasterConfig()
	masterConfig.ID = "master1"
	masterConfig.LocalAddress = 1   // Master address
	masterConfig.RemoteAddress = 10 // Outstation address
	masterConfig.ResponseTimeout = 5 * time.Second

	// Enable unsolicited responses for Class 1 (binary) and Class 2 (analog)
	masterConfig.DisableUnsolOnStartup = false
	masterConfig.UnsolClassMask = app.Class1 | app.Class2

	// Create master callbacks
	callbacks := &MyMasterCallbacks{}

	// Create master
	master, err := dnp3Channel.AddMaster(masterConfig, callbacks)
	if err != nil {
		fmt.Printf("Failed to create master: %v\n", err)
		return
	}

	// Enable master
	if err := master.Enable(); err != nil {
		fmt.Printf("Failed to enable master: %v\n", err)
		return
	}

	fmt.Println("Master enabled")
	fmt.Println("=====================================")
	fmt.Println("Monitoring outstation data...")
	fmt.Println("Unsolicited responses enabled for:")
	fmt.Println("  - Class 1 (Binary inputs)")
	fmt.Println("  - Class 2 (Analog inputs)")
	fmt.Println("Periodic scans also running...")
	fmt.Println("=====================================")

	// Wait a moment for connection to establish
	time.Sleep(1 * time.Second)

	// Perform initial integrity scan
	if err := master.ScanIntegrity(); err != nil {
		fmt.Printf("Initial integrity scan failed: %v\n", err)
	}

	// Add periodic integrity scan (every 30 seconds to get full data refresh)
	handle, err := master.AddIntegrityScan(30 * time.Second)
	if err != nil {
		fmt.Printf("Failed to add periodic scan: %v\n", err)
	} else {
		defer handle.Remove()
	}

	// Add periodic class 1 scan (every 6 seconds to poll for changes)
	class1Handle, err := master.AddClassScan(dnp3.Class1, 60*time.Second)
	if err != nil {
		fmt.Printf("Failed to add class 1 scan: %v\n", err)
	} else {
		defer class1Handle.Remove()
	}

	// Show statistics periodically (every 60 seconds)
	go func() {
		time.Sleep(60 * time.Second) // Wait before first stats
		for {
			stats := tcpChannel.Statistics()
			fmt.Printf("\n=== Connection Statistics ===\n")
			fmt.Printf("  Bytes Sent: %d\n", stats.BytesSent)
			fmt.Printf("  Bytes Received: %d\n", stats.BytesReceived)
			fmt.Printf("  Connects: %d\n", stats.Connects)
			fmt.Printf("  Disconnects: %d\n", stats.Disconnects)
			if stats.ReadErrors > 0 || stats.WriteErrors > 0 {
				fmt.Printf("  Read Errors: %d\n", stats.ReadErrors)
				fmt.Printf("  Write Errors: %d\n", stats.WriteErrors)
			}
			fmt.Println("============================\n")
			time.Sleep(60 * time.Second)
		}
	}()

	// Keep the program running
	fmt.Println("\nMaster running. Press Ctrl+C to exit.")
	select {} // Block forever
}
