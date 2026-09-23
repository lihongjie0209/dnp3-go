package outstation

import (
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// OutstationConfig configures an outstation session
type OutstationConfig struct {
	ID                    string
	LocalAddress          uint16
	RemoteAddress         uint16
	Database              DatabaseConfig
	MaxBinaryEvents       uint
	MaxAnalogEvents       uint
	MaxCounterEvents      uint
	MaxDoubleBitEvents    uint
	AllowUnsolicited      bool
	UnsolConfirmTimeout   time.Duration
	SelectTimeout         time.Duration
	MaxControlsPerRequest uint
	LocalControl          bool
	DeviceTrouble         bool
	MaxRxFragSize         uint16
	MaxTxFragSize         uint16
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

// Point config types
type BinaryPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
}

type DoubleBitBinaryPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
}

type AnalogPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
	Deadband        float64
}

type CounterPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
	Deadband        uint32
}

type FrozenCounterPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
}

type BinaryOutputStatusPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
}

type AnalogOutputStatusPointConfig struct {
	StaticVariation uint8
	EventVariation  uint8
	Class           uint8
	Deadband        float64
}

// OutstationCallbacks defines application callbacks for outstation
type OutstationCallbacks interface {
	CommandHandler
	OnConfirmReceived(unsolicited bool, numClass1, numClass2, numClass3 uint)
	OnUnsolicitedResponse(success bool, seq uint8)
	GetApplicationIIN() types.IIN
}

// CommandHandler processes commands from master
type CommandHandler interface {
	Begin()
	End()

	SelectCROB(crob types.CROB, index uint16) types.CommandStatus
	OperateCROB(crob types.CROB, index uint16, opType OperateType, handler UpdateHandler) types.CommandStatus

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
	EventModeDetect EventMode = iota
	EventModeForce
	EventModeSuppress
)

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
