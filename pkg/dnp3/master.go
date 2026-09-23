package dnp3

import (
	"errors"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

var ErrNotImplemented = errors.New("not yet implemented")

// Master represents a DNP3 master (client) session
type Master interface {
	// Scanning operations
	AddIntegrityScan(period time.Duration) (ScanHandle, error)
	AddClassScan(classes app.ClassField, period time.Duration) (ScanHandle, error)
	AddRangeScan(objGroup, variation uint8, start, stop uint16, period time.Duration) (ScanHandle, error)

	// One-time operations
	ScanIntegrity() error
	ScanClasses(classes app.ClassField) error
	ScanRange(objGroup, variation uint8, start, stop uint16) error

	// Command operations
	SelectAndOperate(commands []types.Command) ([]types.CommandStatus, error)
	DirectOperate(commands []types.Command) ([]types.CommandStatus, error)

	// Control
	Enable() error
	Disable() error
	Shutdown() error
}

// MasterCallbacks defines application callbacks for master
type MasterCallbacks interface {
	SOEHandler

	// OnReceiveIIN is called when IIN bits are received
	OnReceiveIIN(iin types.IIN)

	// OnTaskStart is called when a task starts
	OnTaskStart(taskType TaskType, id int)

	// OnTaskComplete is called when a task completes
	OnTaskComplete(taskType TaskType, id int, result TaskResult)

	// GetTime returns the current time for time synchronization
	GetTime() time.Time
}

// SOEHandler processes measurement data (Sequence of Events)
type SOEHandler interface {
	// Fragment callbacks
	OnBeginFragment(info ResponseInfo)
	OnEndFragment(info ResponseInfo)

	// Measurement processing
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
	Unsolicited bool // True if unsolicited response
	FIR         bool // First fragment
	FIN         bool // Final fragment
}

// HeaderInfo contains information about an object header
type HeaderInfo struct {
	Group     uint8 // Object group
	Variation uint8 // Object variation
	Qualifier uint8 // Qualifier code
	IsEvent   bool  // True if event data
}

// ScanHandle allows control of periodic scans
type ScanHandle interface {
	Demand() error // Trigger scan immediately
	Remove() error // Stop and remove scan
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

// MasterConfig configures a master session
type MasterConfig struct {
	// Identity
	ID string

	// Link layer
	LocalAddress  uint16
	RemoteAddress uint16

	// Timeouts
	ResponseTimeout  time.Duration // Default: 5s
	TaskRetryPeriod  time.Duration // Default: 5s
	TaskStartTimeout time.Duration // Default: 10s

	// Behavior
	DisableUnsolOnStartup bool           // Disable unsolicited on startup
	IgnoreRestartIIN      bool           // Ignore restart IIN bit
	UnsolClassMask        app.ClassField // Classes to accept unsolicited
	StartupIntegrityScan  bool           // Perform integrity scan on startup

	// Timing
	IntegrityPeriod time.Duration // 0 = no automatic integrity scans

	// Advanced
	MaxRxFragSize uint16 // Default: 2048
	MaxTxFragSize uint16 // Default: 2048
}

// DefaultMasterConfig returns a master config with default values
func DefaultMasterConfig() MasterConfig {
	return MasterConfig{
		ResponseTimeout:       5 * time.Second,
		TaskRetryPeriod:       5 * time.Second,
		TaskStartTimeout:      10 * time.Second,
		DisableUnsolOnStartup: true,
		UnsolClassMask:        app.ClassAll,
		StartupIntegrityScan:  true,
		MaxRxFragSize:         2048,
		MaxTxFragSize:         2048,
	}
}
