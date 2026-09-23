package outstation

import (
	"math"
	"sync"

	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Database stores measurement points and generates events
type Database struct {
	// Point arrays
	binary        []BinaryPoint
	doubleBit     []DoubleBitBinaryPoint
	analog        []AnalogPoint
	counter       []CounterPoint
	frozenCounter []FrozenCounterPoint
	binaryOutput  []BinaryOutputStatusPoint
	analogOutput  []AnalogOutputStatusPoint

	// Event buffer
	eventBuffer *EventBuffer

	mu sync.RWMutex
}

// Point types include current value and configuration

// BinaryPoint stores a binary input point
type BinaryPoint struct {
	value           types.Binary
	staticVariation uint8
	eventVariation  uint8
	class           uint8
}

// DoubleBitBinaryPoint stores a double-bit binary input point
type DoubleBitBinaryPoint struct {
	value           types.DoubleBitBinary
	staticVariation uint8
	eventVariation  uint8
	class           uint8
}

// AnalogPoint stores an analog input point
type AnalogPoint struct {
	value           types.Analog
	staticVariation uint8
	eventVariation  uint8
	class           uint8
	deadband        float64
}

// CounterPoint stores a counter point
type CounterPoint struct {
	value           types.Counter
	staticVariation uint8
	eventVariation  uint8
	class           uint8
	deadband        uint32
}

// FrozenCounterPoint stores a frozen counter point
type FrozenCounterPoint struct {
	value           types.FrozenCounter
	staticVariation uint8
	eventVariation  uint8
	class           uint8
}

// BinaryOutputStatusPoint stores a binary output status point
type BinaryOutputStatusPoint struct {
	value           types.BinaryOutputStatus
	staticVariation uint8
	eventVariation  uint8
	class           uint8
}

// AnalogOutputStatusPoint stores an analog output status point
type AnalogOutputStatusPoint struct {
	value           types.AnalogOutputStatus
	staticVariation uint8
	eventVariation  uint8
	class           uint8
	deadband        float64
}

// NewDatabase creates a new database from configuration
func NewDatabase(config DatabaseConfig, eventBuffer *EventBuffer) *Database {
	db := &Database{
		binary:        make([]BinaryPoint, len(config.Binary)),
		doubleBit:     make([]DoubleBitBinaryPoint, len(config.DoubleBit)),
		analog:        make([]AnalogPoint, len(config.Analog)),
		counter:       make([]CounterPoint, len(config.Counter)),
		frozenCounter: make([]FrozenCounterPoint, len(config.FrozenCounter)),
		binaryOutput:  make([]BinaryOutputStatusPoint, len(config.BinaryOutput)),
		analogOutput:  make([]AnalogOutputStatusPoint, len(config.AnalogOutput)),
		eventBuffer:   eventBuffer,
	}

	// Initialize binary points
	for i, cfg := range config.Binary {
		db.binary[i] = BinaryPoint{
			value: types.Binary{
				Value: false,
				Flags: 0,
				Time:  types.ZeroTime(),
			},
			staticVariation: cfg.StaticVariation,
			eventVariation:  cfg.EventVariation,
			class:           cfg.Class,
		}
	}

	// Initialize analog points
	for i, cfg := range config.Analog {
		db.analog[i] = AnalogPoint{
			value: types.Analog{
				Value: 0.0,
				Flags: 0,
				Time:  types.ZeroTime(),
			},
			staticVariation: cfg.StaticVariation,
			eventVariation:  cfg.EventVariation,
			class:           cfg.Class,
			deadband:        cfg.Deadband,
		}
	}

	// Initialize counter points
	for i, cfg := range config.Counter {
		db.counter[i] = CounterPoint{
			value: types.Counter{
				Value: 0,
				Flags: 0,
				Time:  types.ZeroTime(),
			},
			staticVariation: cfg.StaticVariation,
			eventVariation:  cfg.EventVariation,
			class:           cfg.Class,
			deadband:        cfg.Deadband,
		}
	}

	// Initialize other point types similarly...

	return db
}

// UpdateBinary updates a binary point
func (db *Database) UpdateBinary(index uint16, value types.Binary, mode EventMode) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if int(index) >= len(db.binary) {
		return
	}

	point := &db.binary[index]
	oldValue := point.value.Value

	// Update value
	point.value = value

	// Check if event should be generated
	shouldGenerate := false
	switch mode {
	case EventModeForce:
		shouldGenerate = true
	case EventModeDetect:
		shouldGenerate = (oldValue != value.Value)
	case EventModeSuppress:
		shouldGenerate = false
	}

	if shouldGenerate && point.class > 0 {
		db.eventBuffer.AddBinaryEvent(index, value, point.class)
	}
}

// UpdateAnalog updates an analog point
func (db *Database) UpdateAnalog(index uint16, value types.Analog, mode EventMode) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if int(index) >= len(db.analog) {
		return
	}

	point := &db.analog[index]
	oldValue := point.value.Value

	// Update value
	point.value = value

	// Check if event should be generated
	shouldGenerate := false
	switch mode {
	case EventModeForce:
		shouldGenerate = true
	case EventModeDetect:
		// Check deadband
		shouldGenerate = math.Abs(oldValue-value.Value) > point.deadband
	case EventModeSuppress:
		shouldGenerate = false
	}

	if shouldGenerate && point.class > 0 {
		db.eventBuffer.AddAnalogEvent(index, value, point.class)
	}
}

// UpdateCounter updates a counter point
func (db *Database) UpdateCounter(index uint16, value types.Counter, mode EventMode) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if int(index) >= len(db.counter) {
		return
	}

	point := &db.counter[index]
	oldValue := point.value.Value

	// Update value
	point.value = value

	// Check if event should be generated
	shouldGenerate := false
	switch mode {
	case EventModeForce:
		shouldGenerate = true
	case EventModeDetect:
		// Check deadband
		var diff uint32
		if value.Value > oldValue {
			diff = value.Value - oldValue
		} else {
			diff = oldValue - value.Value
		}
		shouldGenerate = diff > point.deadband
	case EventModeSuppress:
		shouldGenerate = false
	}

	if shouldGenerate && point.class > 0 {
		db.eventBuffer.AddCounterEvent(index, value, point.class)
	}
}

// GetBinary returns a binary point value
func (db *Database) GetBinary(index uint16) (types.Binary, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if int(index) >= len(db.binary) {
		return types.Binary{}, false
	}

	return db.binary[index].value, true
}

// GetAnalog returns an analog point value
func (db *Database) GetAnalog(index uint16) (types.Analog, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if int(index) >= len(db.analog) {
		return types.Analog{}, false
	}

	return db.analog[index].value, true
}

// GetCounter returns a counter point value
func (db *Database) GetCounter(index uint16) (types.Counter, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if int(index) >= len(db.counter) {
		return types.Counter{}, false
	}

	return db.counter[index].value, true
}

// BinaryCount returns the number of binary points
func (db *Database) BinaryCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.binary)
}

// AnalogCount returns the number of analog points
func (db *Database) AnalogCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.analog)
}

// CounterCount returns the number of counter points
func (db *Database) CounterCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.counter)
}
