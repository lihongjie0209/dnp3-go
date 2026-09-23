package master

import (
	"errors"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Scan operations

// AddIntegrityScan adds a periodic integrity scan
func (m *master) AddIntegrityScan(period time.Duration) (ScanHandle, error) {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	task := &IntegrityScanTask{
		id:       m.nextScanID,
		priority: PriorityNormal,
	}

	scan := &PeriodicScan{
		id:      m.nextScanID,
		task:    task,
		period:  period,
		nextRun: time.Now(),
		enabled: true,
	}

	m.scans[m.nextScanID] = scan
	m.nextScanID++

	// Schedule first run
	m.taskQueue.Push(task, task.Priority(), time.Now())

	m.logger.Info("Master %s: Added integrity scan (period=%s, id=%d)", m.config.ID, period, scan.id)

	return &ScanHandleImpl{id: scan.id, master: m}, nil
}

// AddClassScan adds a periodic class scan
func (m *master) AddClassScan(classes app.ClassField, period time.Duration) (ScanHandle, error) {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	task := &ClassScanTask{
		id:       m.nextScanID,
		classes:  classes,
		priority: PriorityNormal,
	}

	scan := &PeriodicScan{
		id:      m.nextScanID,
		task:    task,
		period:  period,
		nextRun: time.Now(),
		enabled: true,
	}

	m.scans[m.nextScanID] = scan
	m.nextScanID++

	// Schedule first run
	m.taskQueue.Push(task, task.Priority(), time.Now())

	m.logger.Info("Master %s: Added class scan %s (period=%s, id=%d)", m.config.ID, classes, period, scan.id)

	return &ScanHandleImpl{id: scan.id, master: m}, nil
}

// AddRangeScan adds a periodic range scan
func (m *master) AddRangeScan(objGroup, variation uint8, start, stop uint16, period time.Duration) (ScanHandle, error) {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	task := &RangeScanTask{
		id:        m.nextScanID,
		group:     objGroup,
		variation: variation,
		start:     start,
		stop:      stop,
		priority:  PriorityNormal,
	}

	scan := &PeriodicScan{
		id:      m.nextScanID,
		task:    task,
		period:  period,
		nextRun: time.Now(),
		enabled: true,
	}

	m.scans[m.nextScanID] = scan
	m.nextScanID++

	// Schedule first run
	m.taskQueue.Push(task, task.Priority(), time.Now())

	m.logger.Info("Master %s: Added range scan G%dV%d [%d-%d] (period=%s, id=%d)",
		m.config.ID, objGroup, variation, start, stop, period, scan.id)

	return &ScanHandleImpl{id: scan.id, master: m}, nil
}

// ScanIntegrity performs one-time integrity scan
func (m *master) ScanIntegrity() error {
	task := &IntegrityScanTask{
		id:       0,
		priority: PriorityHigh,
	}
	m.taskQueue.Push(task, task.Priority(), time.Now())
	return nil
}

// ScanClasses performs one-time class scan
func (m *master) ScanClasses(classes app.ClassField) error {
	task := &ClassScanTask{
		id:       0,
		classes:  classes,
		priority: PriorityHigh,
	}
	m.taskQueue.Push(task, task.Priority(), time.Now())
	return nil
}

// ScanRange performs one-time range scan
func (m *master) ScanRange(objGroup, variation uint8, start, stop uint16) error {
	task := &RangeScanTask{
		id:        0,
		group:     objGroup,
		variation: variation,
		start:     start,
		stop:      stop,
		priority:  PriorityHigh,
	}
	m.taskQueue.Push(task, task.Priority(), time.Now())
	return nil
}

// demandScan triggers an immediate scan
func (m *master) demandScan(id int) error {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	scan, exists := m.scans[id]
	if !exists {
		return errors.New("scan not found")
	}

	scan.demanded = true
	m.logger.Info("Master %s: Scan %d demanded", m.config.ID, id)
	return nil
}

// removeScan removes a periodic scan
func (m *master) removeScan(id int) error {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	delete(m.scans, id)
	m.logger.Info("Master %s: Scan %d removed", m.config.ID, id)
	return nil
}

// performIntegrityScan performs an integrity scan (Class 0) using app layer helpers
func (m *master) performIntegrityScan() error {
	apdu := app.BuildIntegrityPollRequest(m.getNextSequence())

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performClassScan performs a class scan using app layer helpers
func (m *master) performClassScan(classes app.ClassField) error {
	// Build objects for specified classes
	objects := app.BuildClassRead(classes)
	apdu := app.BuildReadRequest(m.getNextSequence(), objects)

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performRangeScan performs a range scan using app layer helpers
func (m *master) performRangeScan(group, variation uint8, start, stop uint16) error {
	// Use app layer helper - automatically selects optimal qualifier
	objects := app.BuildRangeRead(group, variation, uint32(start), uint32(stop))
	apdu := app.BuildReadRequest(m.getNextSequence(), objects)

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// Command operations

// SelectAndOperate performs SELECT then OPERATE
func (m *master) SelectAndOperate(commands []types.Command) ([]types.CommandStatus, error) {
	task := &CommandTask{
		commands:     commands,
		selectBefore: true,
		priority:     PriorityHigh,
		result:       make(chan CommandResult, 1),
	}

	m.taskQueue.Push(task, task.Priority(), time.Now())

	// Wait for result
	select {
	case result := <-task.result:
		return result.Statuses, result.Error
	case <-time.After(m.config.ResponseTimeout * 2): // SELECT + OPERATE
		return nil, ErrTimeout
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

// DirectOperate performs DIRECT OPERATE
func (m *master) DirectOperate(commands []types.Command) ([]types.CommandStatus, error) {
	task := &CommandTask{
		commands:     commands,
		selectBefore: false,
		priority:     PriorityHigh,
		result:       make(chan CommandResult, 1),
	}

	m.taskQueue.Push(task, task.Priority(), time.Now())

	// Wait for result
	select {
	case result := <-task.result:
		return result.Statuses, result.Error
	case <-time.After(m.config.ResponseTimeout):
		return nil, ErrTimeout
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

// performSelectAndOperate executes SELECT and OPERATE using app layer helpers
func (m *master) performSelectAndOperate(commands []types.Command) ([]types.CommandStatus, error) {
	// Build command objects using app layer helpers
	objects := buildCROBObjects(commands)

	// SELECT phase
	selectAPDU := app.BuildSelectRequest(m.getNextSequence(), objects)
	selectResp, err := m.sendAndWait(selectAPDU, m.config.ResponseTimeout)
	if err != nil {
		return nil, err
	}

	// Parse SELECT response status
	statuses := parseCommandResponse(selectResp, len(commands))
	if !allSuccess(statuses) {
		return statuses, nil
	}

	// OPERATE phase with same objects
	operateAPDU := app.BuildOperateRequest(m.getNextSequence(), objects)
	operateResp, err := m.sendAndWait(operateAPDU, m.config.ResponseTimeout)
	if err != nil {
		return nil, err
	}

	// Parse OPERATE response status
	return parseCommandResponse(operateResp, len(commands)), nil
}

// performDirectOperate executes DIRECT OPERATE using app layer helpers
func (m *master) performDirectOperate(commands []types.Command) ([]types.CommandStatus, error) {
	objects := buildCROBObjects(commands)

	apdu := app.BuildDirectOperateRequest(m.getNextSequence(), objects)
	resp, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	if err != nil {
		return nil, err
	}

	return parseCommandResponse(resp, len(commands)), nil
}

// buildCROBObjects builds CROB objects from commands using app layer helpers
func buildCROBObjects(commands []types.Command) []byte {
	builder := app.NewObjectBuilder()

	for _, cmd := range commands {
		// Extract CROB from command data
		crobData, ok := cmd.Data.(types.CROB)
		if !ok {
			continue // Skip non-CROB commands
		}

		// Convert types.CROB to app.CROB
		var crob app.CROB

		switch crobData.OpType {
		case types.ControlCodeLatchOn:
			crob = app.NewLatchOn()
		case types.ControlCodeLatchOff:
			crob = app.NewLatchOff()
		case types.ControlCodePulseOn:
			crob = app.NewPulseOn(crobData.OnTimeMs)
		case types.ControlCodePulseOff:
			crob = app.NewPulseOff(crobData.OffTimeMs)
		default:
			crob = app.NewCROB(uint8(crobData.OpType), crobData.Count, crobData.OnTimeMs, crobData.OffTimeMs)
		}

		// Add object header for this command
		builder.AddHeader(
			app.GroupBinaryOutputCommand,
			1, // CROB variation
			app.Qualifier8BitStartStop,
			app.StartStopRange{Start: uint32(cmd.Index), Stop: uint32(cmd.Index)},
		)

		// Add CROB data
		builder.AddRawData(crob.Serialize())
	}

	return builder.Build()
}

// parseCommandResponse parses command response and extracts status codes
func parseCommandResponse(apdu *app.APDU, numCommands int) []types.CommandStatus {
	statuses := make([]types.CommandStatus, numCommands)

	if apdu == nil || len(apdu.Objects) == 0 {
		// No response data, assume success
		for i := range statuses {
			statuses[i] = types.CommandStatusSuccess
		}
		return statuses
	}

	// Parse response objects to extract status codes
	parser := app.NewParser(apdu.Objects)
	cmdIndex := 0

	for parser.HasMore() && cmdIndex < numCommands {
		header, err := parser.ReadObjectHeader()
		if err != nil {
			break
		}

		count := app.GetCount(header.Range)

		// Read CROB responses
		for i := uint32(0); i < count && cmdIndex < numCommands; i++ {
			data, err := parser.ReadBytes(11) // CROB is 11 bytes
			if err != nil {
				break
			}

			crob, err := app.ParseCROB(data)
			if err != nil {
				statuses[cmdIndex] = types.CommandStatusFormatError
			} else {
				statuses[cmdIndex] = types.CommandStatus(crob.Status)
			}
			cmdIndex++
		}
	}

	// Fill remaining with success if not enough responses
	for i := cmdIndex; i < numCommands; i++ {
		statuses[i] = types.CommandStatusSuccess
	}

	return statuses
}

// allSuccess checks if all command statuses are successful
func allSuccess(statuses []types.CommandStatus) bool {
	for _, status := range statuses {
		if status != types.CommandStatusSuccess {
			return false
		}
	}
	return true
}

// performEnableUnsolicited enables unsolicited responses using app layer helpers
func (m *master) performEnableUnsolicited(classes app.ClassField) error {
	apdu := app.BuildEnableUnsolicitedRequest(m.getNextSequence(), classes)

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performDisableUnsolicited disables unsolicited responses using app layer helpers
func (m *master) performDisableUnsolicited(classes app.ClassField) error {
	apdu := app.BuildDisableUnsolicitedRequest(m.getNextSequence(), classes)

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performTimeSync performs time synchronization using app layer helpers
func (m *master) performTimeSync() error {
	apdu := app.BuildTimeSyncNowRequest(m.getNextSequence())

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performColdRestart performs cold restart using app layer helpers
func (m *master) performColdRestart() error {
	apdu := app.BuildColdRestartRequest(m.getNextSequence())

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performWarmRestart performs warm restart using app layer helpers
func (m *master) performWarmRestart() error {
	apdu := app.BuildWarmRestartRequest(m.getNextSequence())

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}
