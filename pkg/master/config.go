package master

import (
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// MasterConfig configures a master session
type MasterConfig struct {
	// Identity
	ID string

	// Link layer
	LocalAddress  uint16
	RemoteAddress uint16

	// Timeouts
	ResponseTimeout  time.Duration
	TaskRetryPeriod  time.Duration
	TaskStartTimeout time.Duration

	// Behavior
	DisableUnsolOnStartup bool
	IgnoreRestartIIN      bool
	UnsolClassMask        app.ClassField
	StartupIntegrityScan  bool

	// Timing
	IntegrityPeriod time.Duration

	// Advanced
	MaxRxFragSize uint16
	MaxTxFragSize uint16
}

// MasterCallbacks defines application callbacks for master
type MasterCallbacks interface {
	SOEHandler

	OnReceiveIIN(iin types.IIN)
	OnTaskStart(taskType TaskType, id int)
	OnTaskComplete(taskType TaskType, id int, result TaskResult)
	GetTime() time.Time
}

// SOEHandler processes measurement data
type SOEHandler interface {
	OnBeginFragment(info ResponseInfo)
	OnEndFragment(info ResponseInfo)

	ProcessBinary(info HeaderInfo, values []types.IndexedBinary)
	ProcessDoubleBitBinary(info HeaderInfo, values []types.IndexedDoubleBitBinary)
	ProcessAnalog(info HeaderInfo, values []types.IndexedAnalog)
	ProcessCounter(info HeaderInfo, values []types.IndexedCounter)
	ProcessFrozenCounter(info HeaderInfo, values []types.IndexedFrozenCounter)
	ProcessBinaryOutputStatus(info HeaderInfo, values []types.IndexedBinaryOutputStatus)
	ProcessAnalogOutputStatus(info HeaderInfo, values []types.IndexedAnalogOutputStatus)
}

// ResponseInfo contains information about a response fragment
type ResponseInfo struct {
	Unsolicited bool
	FIR         bool
	FIN         bool
}

// HeaderInfo contains information about an object header
type HeaderInfo struct {
	Group     uint8
	Variation uint8
	Qualifier uint8
	IsEvent   bool
}

// TaskType identifies the type of master task
type TaskType int

const (
	TaskTypeIntegrityScan TaskType = iota
	TaskTypeClassScan
	TaskTypeRangeScan
	TaskTypeCommand
)

// TaskResult indicates the result of a task
type TaskResult int

const (
	TaskResultSuccess TaskResult = iota
	TaskResultFailure
	TaskResultTimeout
)

// ScanHandle allows control of periodic scans
type ScanHandle interface {
	Demand() error
	Remove() error
}
