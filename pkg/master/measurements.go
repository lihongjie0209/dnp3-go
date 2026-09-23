package master

import (
	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// processMeasurements processes measurement data from response
func (m *master) processMeasurements(apdu *app.APDU) {
	info := ResponseInfo{
		Unsolicited: apdu.FunctionCode == app.FuncUnsolicitedResponse,
		FIR:         apdu.FIR,
		FIN:         apdu.FIN,
	}

	m.callbacks.OnBeginFragment(info)

	parser := app.NewParser(apdu.Objects)

	for parser.HasMore() {
		header, err := parser.ReadObjectHeader()
		if err != nil {
			m.logger.Error("Master %s: Failed to parse object header: %v", m.config.ID, err)
			break
		}

		// Validate object header using app layer helper
		if err := app.ValidateObjectHeader(header); err != nil {
			m.logger.Warn("Master %s: Invalid object header: %v", m.config.ID, err)
			continue
		}

		headerInfo := HeaderInfo{
			Group:     header.Group,
			Variation: header.Variation,
			Qualifier: uint8(header.Qualifier),
			IsEvent:   isEventGroup(header.Group),
		}

		// Process based on group
		switch header.Group {
		case app.GroupBinaryInput, app.GroupBinaryInputEvent:
			m.processBinaryObjects(parser, header, headerInfo)

		case app.GroupAnalogInput, app.GroupAnalogInputEvent:
			m.processAnalogObjects(parser, header, headerInfo)

		case app.GroupCounter, app.GroupCounterEvent:
			m.processCounterObjects(parser, header, headerInfo)

		case app.GroupBinaryOutput, app.GroupBinaryOutputEvent:
			m.processBinaryOutputStatus(parser, header, headerInfo)

		case app.GroupAnalogOutputStatus, app.GroupAnalogOutputEvent:
			m.processAnalogOutputStatus(parser, header, headerInfo)

		default:
			// Skip unknown group using app layer helper
			count := app.GetCount(header.Range)
			objectSize := app.GetObjectSize(header.Group, header.Variation)
			if objectSize > 0 {
				parser.Skip(int(count) * objectSize)
			}
			m.logger.Debug("Master %s: Skipped unknown group G%dV%d",
				m.config.ID, header.Group, header.Variation)
		}
	}

	m.callbacks.OnEndFragment(info)
}

// processBinaryObjects processes binary input objects using app layer parsers
func (m *master) processBinaryObjects(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	count := app.GetCount(header.Range)
	values := make([]types.IndexedBinary, 0, count)

	// Use app layer helper for object size
	objectSize := app.GetObjectSize(header.Group, header.Variation)
	if objectSize == 0 {
		m.logger.Warn("Master %s: Unknown size for G%dV%d", m.config.ID, header.Group, header.Variation)
		return
	}

	// Determine starting index
	startIndex := uint32(0)
	if r, ok := header.Range.(app.StartStopRange); ok {
		startIndex = r.Start
	}

	// Parse each object using app layer helpers
	for i := uint32(0); i < count; i++ {
		data, err := parser.ReadBytes(objectSize)
		if err != nil {
			m.logger.Error("Master %s: Failed to read binary object: %v", m.config.ID, err)
			break
		}

		// Parse using app layer helper
		bi := app.ParseBinaryInput(data)

		// Convert to indexed value with proper wrapper type
		value := types.IndexedBinary{
			Index: uint16(startIndex + i),
			Value: types.Binary{
				Value: bi.Value,
				Flags: types.Flags(bi.Flags),
			},
		}

		values = append(values, value)
	}

	m.callbacks.ProcessBinary(info, values)
}

// processAnalogObjects processes analog input objects using app layer parsers
func (m *master) processAnalogObjects(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	count := app.GetCount(header.Range)
	values := make([]types.IndexedAnalog, 0, count)

	objectSize := app.GetObjectSize(header.Group, header.Variation)
	if objectSize == 0 {
		m.logger.Warn("Master %s: Unknown size for G%dV%d", m.config.ID, header.Group, header.Variation)
		return
	}

	startIndex := uint32(0)
	if r, ok := header.Range.(app.StartStopRange); ok {
		startIndex = r.Start
	}

	for i := uint32(0); i < count; i++ {
		data, err := parser.ReadBytes(objectSize)
		if err != nil {
			m.logger.Error("Master %s: Failed to read analog object: %v", m.config.ID, err)
			break
		}

		// Parse based on variation using app layer helpers
		var ai app.AnalogInput
		var analogValue float64

		switch header.Variation {
		case app.AnalogInput32Bit:
			ai = app.ParseAnalogInput32Bit(data)
			if val, ok := ai.Value.(int32); ok {
				analogValue = float64(val)
			}
		case app.AnalogInput16Bit:
			ai = app.ParseAnalogInput16Bit(data)
			if val, ok := ai.Value.(int16); ok {
				analogValue = float64(val)
			}
		case app.AnalogInputFloat:
			ai = app.ParseAnalogInputFloat(data)
			if val, ok := ai.Value.(float32); ok {
				analogValue = float64(val)
			}
		case app.AnalogInputDouble:
			ai = app.ParseAnalogInputDouble(data)
			if val, ok := ai.Value.(float64); ok {
				analogValue = val
			}
		default:
			// Try 32-bit as default
			ai = app.ParseAnalogInput32Bit(data)
			if val, ok := ai.Value.(int32); ok {
				analogValue = float64(val)
			}
		}

		value := types.IndexedAnalog{
			Index: uint16(startIndex + i),
			Value: types.Analog{
				Value: analogValue,
				Flags: types.Flags(ai.Flags),
			},
		}

		values = append(values, value)
	}

	m.callbacks.ProcessAnalog(info, values)
}

// processCounterObjects processes counter objects using app layer parsers
func (m *master) processCounterObjects(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	count := app.GetCount(header.Range)
	values := make([]types.IndexedCounter, 0, count)

	objectSize := app.GetObjectSize(header.Group, header.Variation)
	if objectSize == 0 {
		m.logger.Warn("Master %s: Unknown size for G%dV%d", m.config.ID, header.Group, header.Variation)
		return
	}

	startIndex := uint32(0)
	if r, ok := header.Range.(app.StartStopRange); ok {
		startIndex = r.Start
	}

	for i := uint32(0); i < count; i++ {
		data, err := parser.ReadBytes(objectSize)
		if err != nil {
			m.logger.Error("Master %s: Failed to read counter object: %v", m.config.ID, err)
			break
		}

		// Parse using app layer helpers
		var counter app.Counter

		switch header.Variation {
		case app.Counter32Bit, app.Counter32BitWithFlag:
			counter = app.ParseCounter32Bit(data)
		case app.Counter16Bit, app.Counter16BitWithFlag:
			counter = app.ParseCounter16Bit(data)
		default:
			counter = app.ParseCounter32Bit(data)
		}

		value := types.IndexedCounter{
			Index: uint16(startIndex + i),
			Value: types.Counter{
				Value: counter.Value,
				Flags: types.Flags(counter.Flags),
			},
		}

		values = append(values, value)
	}

	m.callbacks.ProcessCounter(info, values)
}

// processBinaryOutputStatus processes binary output status using app layer parsers
func (m *master) processBinaryOutputStatus(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	count := app.GetCount(header.Range)
	values := make([]types.IndexedBinaryOutputStatus, 0, count)

	objectSize := app.GetObjectSize(header.Group, header.Variation)
	if objectSize == 0 {
		objectSize = 1 // Binary output is 1 byte
	}

	startIndex := uint32(0)
	if r, ok := header.Range.(app.StartStopRange); ok {
		startIndex = r.Start
	}

	for i := uint32(0); i < count; i++ {
		data, err := parser.ReadBytes(objectSize)
		if err != nil {
			m.logger.Error("Master %s: Failed to read binary output: %v", m.config.ID, err)
			break
		}

		bo := app.ParseBinaryOutput(data)

		value := types.IndexedBinaryOutputStatus{
			Index: uint16(startIndex + i),
			Value: types.BinaryOutputStatus{
				Value: bo.Value,
				Flags: types.Flags(bo.Flags),
			},
		}

		values = append(values, value)
	}

	m.callbacks.ProcessBinaryOutputStatus(info, values)
}

// processAnalogOutputStatus processes analog output status using app layer parsers
func (m *master) processAnalogOutputStatus(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	count := app.GetCount(header.Range)
	values := make([]types.IndexedAnalogOutputStatus, 0, count)

	objectSize := app.GetObjectSize(header.Group, header.Variation)
	if objectSize == 0 {
		m.logger.Warn("Master %s: Unknown size for G%dV%d", m.config.ID, header.Group, header.Variation)
		return
	}

	startIndex := uint32(0)
	if r, ok := header.Range.(app.StartStopRange); ok {
		startIndex = r.Start
	}

	for i := uint32(0); i < count; i++ {
		data, err := parser.ReadBytes(objectSize)
		if err != nil {
			m.logger.Error("Master %s: Failed to read analog output: %v", m.config.ID, err)
			break
		}

		// Parse analog output status (Group 40)
		var ai app.AnalogInput
		var analogValue float64

		switch header.Variation {
		case 1: // 32-bit
			ai = app.ParseAnalogInput32Bit(data)
			if val, ok := ai.Value.(int32); ok {
				analogValue = float64(val)
			}
		case 2: // 16-bit
			ai = app.ParseAnalogInput16Bit(data)
			if val, ok := ai.Value.(int16); ok {
				analogValue = float64(val)
			}
		case 3: // Float
			ai = app.ParseAnalogInputFloat(data)
			if val, ok := ai.Value.(float32); ok {
				analogValue = float64(val)
			}
		case 4: // Double
			ai = app.ParseAnalogInputDouble(data)
			if val, ok := ai.Value.(float64); ok {
				analogValue = val
			}
		}

		value := types.IndexedAnalogOutputStatus{
			Index: uint16(startIndex + i),
			Value: types.AnalogOutputStatus{
				Value: analogValue,
				Flags: types.Flags(ai.Flags),
			},
		}

		values = append(values, value)
	}

	m.callbacks.ProcessAnalogOutputStatus(info, values)
}

// isEventGroup returns true if the group is an event group
func isEventGroup(group uint8) bool {
	switch group {
	case app.GroupBinaryInputEvent,
		app.GroupDoubleBitBinaryEvent,
		app.GroupCounterEvent,
		app.GroupFrozenCounterEvent,
		app.GroupAnalogInputEvent,
		app.GroupFrozenAnalogEvent,
		app.GroupBinaryOutputEvent,
		app.GroupAnalogOutputEvent:
		return true
	default:
		return false
	}
}
