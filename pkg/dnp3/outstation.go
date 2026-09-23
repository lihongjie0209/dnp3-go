package dnp3

import (
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Outstation represents a DNP3 outstation (server) session
type Outstation interface {
	// Apply applies measurement updates atomically
	Apply(updates *Updates) error

	// SetConfig updates the outstation configuration
	SetConfig(config OutstationConfig) error

	// Control
	Enable() error
	Disable() error
	Shutdown() error
}

// OutstationCallbacks defines application callbacks for outstation
type OutstationCallbacks interface {
	CommandHandler

	// OnConfirmReceived is called when a confirm is received
	OnConfirmReceived(unsolicited bool, numClass1, numClass2, numClass3 uint)

	// OnUnsolicitedResponse is called when an unsolicited response is sent
	OnUnsolicitedResponse(success bool, seq uint8)

	// GetApplicationIIN returns application-specific IIN bits
	GetApplicationIIN() types.IIN
}

// CommandHandler processes commands from master
type CommandHandler interface {
	// Begin is called at the start of command processing
	Begin()

	// End is called at the end of command processing
	End()

	// CROB commands
	SelectCROB(crob types.CROB, index uint16) types.CommandStatus
	OperateCROB(crob types.CROB, index uint16, opType OperateType, handler UpdateHandler) types.CommandStatus

	// Analog output commands
	SelectAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16) types.CommandStatus
	OperateAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16, opType OperateType, handler UpdateHandler) types.CommandStatus

	SelectAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16) types.CommandStatus
	OperateAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16, opType OperateType, handler UpdateHandler) types.CommandStatus

	SelectAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16) types.CommandStatus
	OperateAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16, opType OperateType, handler UpdateHandler) types.CommandStatus

	SelectAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16) types.CommandStatus
	OperateAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16, opType OperateType, handler UpdateHandler) types.CommandStatus
}

// UpdateHandler allows updating measurements during command processing
type UpdateHandler interface {
	Update(meas interface{}, index uint16, mode EventMode) bool
}

// OperateType indicates the type of operate command
type OperateType int

const (
	OperateTypeSelectBeforeOperate OperateType = iota
	OperateTypeDirectOperate
	OperateTypeDirectOperateNoAck
)

// EventMode controls event generation
type EventMode int

const (
	EventModeDetect   EventMode = iota // Auto-detect based on value change
	EventModeForce                     // Always generate event
	EventModeSuppress                  // Never generate event
)

// Updates represents a batch of measurement updates
// This is a simple wrapper - the actual data is managed internally
type Updates struct {
	// Internal data managed by the builder
	Data interface{}
}

// MeasurementType identifies the type of measurement
type MeasurementType int

const (
	MeasurementTypeBinary MeasurementType = iota
	MeasurementTypeDoubleBitBinary
	MeasurementTypeAnalog
	MeasurementTypeCounter
	MeasurementTypeFrozenCounter
	MeasurementTypeBinaryOutputStatus
	MeasurementTypeAnalogOutputStatus
)

// OutstationConfig configures an outstation session
type OutstationConfig struct {
	// Identity
	ID string

	// Link layer
	LocalAddress  uint16
	RemoteAddress uint16

	// Database
	Database DatabaseConfig

	// Event buffers
	MaxBinaryEvents    uint // Per class
	MaxAnalogEvents    uint
	MaxCounterEvents   uint
	MaxDoubleBitEvents uint

	// Behavior
	AllowUnsolicited      bool          // Allow unsolicited responses
	UnsolConfirmTimeout   time.Duration // Default: 5s
	SelectTimeout         time.Duration // Default: 10s
	MaxControlsPerRequest uint          // Default: 16

	// IIN bits
	LocalControl  bool // IIN1.5
	DeviceTrouble bool // IIN1.6

	// Advanced
	MaxRxFragSize uint16 // Default: 2048
	MaxTxFragSize uint16 // Default: 2048
}

// DatabaseConfig defines point counts and configurations
type DatabaseConfig struct {
	Binary        []BinaryPointConfig
	DoubleBit     []DoubleBitBinaryPointConfig
	Analog        []AnalogPointConfig
	Counter       []CounterPointConfig
	FrozenCounter []FrozenCounterPointConfig
	BinaryOutput  []BinaryOutputStatusPointConfig
	AnalogOutput  []AnalogOutputStatusPointConfig
}

// BinaryPointConfig configures a binary point
type BinaryPointConfig struct {
	StaticVariation uint8 // Default variation for static reads
	EventVariation  uint8 // Default variation for events
	Class           uint8 // Event class (0=none, 1-3)
}

// DoubleBitBinaryPointConfig configures a double-bit binary point
type DoubleBitBinaryPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
}

// AnalogPointConfig configures an analog point
type AnalogPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
	Deadband        float64 // Event generation threshold
}

// CounterPointConfig configures a counter point
type CounterPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
	Deadband        uint32 // Event generation threshold
}

// FrozenCounterPointConfig configures a frozen counter point
type FrozenCounterPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
}

// BinaryOutputStatusPointConfig configures a binary output status point
type BinaryOutputStatusPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
}

// AnalogOutputStatusPointConfig configures an analog output status point
type AnalogOutputStatusPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
	Deadband        float64
}

// DefaultOutstationConfig returns an outstation config with default values
func DefaultOutstationConfig() OutstationConfig {
	return OutstationConfig{
		MaxBinaryEvents:       100,
		MaxAnalogEvents:       100,
		MaxCounterEvents:      100,
		MaxDoubleBitEvents:    100,
		AllowUnsolicited:      true,
		UnsolConfirmTimeout:   5 * time.Second,
		SelectTimeout:         10 * time.Second,
		MaxControlsPerRequest: 16,
		MaxRxFragSize:         2048,
		MaxTxFragSize:         2048,
	}
}
