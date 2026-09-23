package master

import (
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Task represents a master task
type Task interface {
	Execute(m *master) error
	Priority() int
	Type() TaskType
}

// Priority levels
const (
	PriorityHigh   = 100
	PriorityNormal = 50
	PriorityLow    = 10
)

// IntegrityScanTask performs a Class 0 (integrity) scan
type IntegrityScanTask struct {
	id       int
	priority int
}

func (t *IntegrityScanTask) Execute(m *master) error {
	m.logger.Info("Master %s: Executing integrity scan", m.config.ID)
	return m.performIntegrityScan()
}

func (t *IntegrityScanTask) Priority() int {
	return t.priority
}

func (t *IntegrityScanTask) Type() TaskType {
	return TaskTypeIntegrityScan
}

// ClassScanTask performs a class scan
type ClassScanTask struct {
	id       int
	classes  app.ClassField
	priority int
}

func (t *ClassScanTask) Execute(m *master) error {
	m.logger.Info("Master %s: Executing class scan %s", m.config.ID, t.classes)
	return m.performClassScan(t.classes)
}

func (t *ClassScanTask) Priority() int {
	return t.priority
}

func (t *ClassScanTask) Type() TaskType {
	return TaskTypeClassScan
}

// RangeScanTask performs a range scan
type RangeScanTask struct {
	id        int
	group     uint8
	variation uint8
	start     uint16
	stop      uint16
	priority  int
}

func (t *RangeScanTask) Execute(m *master) error {
	m.logger.Info("Master %s: Executing range scan G%dV%d [%d-%d]",
		m.config.ID, t.group, t.variation, t.start, t.stop)
	return m.performRangeScan(t.group, t.variation, t.start, t.stop)
}

func (t *RangeScanTask) Priority() int {
	return t.priority
}

func (t *RangeScanTask) Type() TaskType {
	return TaskTypeRangeScan
}

// CommandTask executes a command
type CommandTask struct {
	commands     []types.Command
	selectBefore bool
	priority     int
	result       chan CommandResult
}

type CommandResult struct {
	Statuses []types.CommandStatus
	Error    error
}

func (t *CommandTask) Execute(m *master) error {
	m.logger.Info("Master %s: Executing command task (%d commands)", m.config.ID, len(t.commands))

	var statuses []types.CommandStatus
	var err error

	if t.selectBefore {
		statuses, err = m.performSelectAndOperate(t.commands)
	} else {
		statuses, err = m.performDirectOperate(t.commands)
	}

	// Send result
	select {
	case t.result <- CommandResult{Statuses: statuses, Error: err}:
	default:
	}

	return err
}

func (t *CommandTask) Priority() int {
	return t.priority
}

func (t *CommandTask) Type() TaskType {
	return TaskTypeCommand
}

// PeriodicScan represents a periodic scan task
type PeriodicScan struct {
	id       int
	task     Task
	period   time.Duration
	nextRun  time.Time
	enabled  bool
	demanded bool
}

// ScanHandleImpl implements ScanHandle
type ScanHandleImpl struct {
	id     int
	master *master
}

func (h *ScanHandleImpl) Demand() error {
	return h.master.demandScan(h.id)
}

func (h *ScanHandleImpl) Remove() error {
	return h.master.removeScan(h.id)
}
