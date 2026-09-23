package outstation

import (
	"container/list"
	"sync"

	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// EventBuffer manages event storage per class
type EventBuffer struct {
	class1 *list.List
	class2 *list.List
	class3 *list.List

	maxSize uint
	mu      sync.RWMutex
}

// Event represents a generic event
type Event struct {
	Index uint16
	Type  EventType
	Value interface{}
	Class uint8
}

// EventType identifies the type of event
type EventType int

const (
	EventTypeBinary EventType = iota
	EventTypeDoubleBitBinary
	EventTypeAnalog
	EventTypeCounter
	EventTypeFrozenCounter
	EventTypeBinaryOutputStatus
	EventTypeAnalogOutputStatus
)

// NewEventBuffer creates a new event buffer
func NewEventBuffer(maxSize uint) *EventBuffer {
	return &EventBuffer{
		class1:  list.New(),
		class2:  list.New(),
		class3:  list.New(),
		maxSize: maxSize,
	}
}

// AddBinaryEvent adds a binary event
func (eb *EventBuffer) AddBinaryEvent(index uint16, value types.Binary, class uint8) {
	event := &Event{
		Index: index,
		Type:  EventTypeBinary,
		Value: value,
		Class: class,
	}
	eb.addEvent(event, class)
}

// AddAnalogEvent adds an analog event
func (eb *EventBuffer) AddAnalogEvent(index uint16, value types.Analog, class uint8) {
	event := &Event{
		Index: index,
		Type:  EventTypeAnalog,
		Value: value,
		Class: class,
	}
	eb.addEvent(event, class)
}

// AddCounterEvent adds a counter event
func (eb *EventBuffer) AddCounterEvent(index uint16, value types.Counter, class uint8) {
	event := &Event{
		Index: index,
		Type:  EventTypeCounter,
		Value: value,
		Class: class,
	}
	eb.addEvent(event, class)
}

// addEvent adds an event to the appropriate class buffer
func (eb *EventBuffer) addEvent(event *Event, class uint8) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	var targetList *list.List
	switch class {
	case 1:
		targetList = eb.class1
	case 2:
		targetList = eb.class2
	case 3:
		targetList = eb.class3
	default:
		return
	}

	// Check capacity
	if uint(targetList.Len()) >= eb.maxSize {
		// Remove oldest event
		targetList.Remove(targetList.Front())
	}

	// Add new event
	targetList.PushBack(event)
}

// GetClass1Count returns the number of Class 1 events
func (eb *EventBuffer) GetClass1Count() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.class1.Len()
}

// GetClass2Count returns the number of Class 2 events
func (eb *EventBuffer) GetClass2Count() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.class2.Len()
}

// GetClass3Count returns the number of Class 3 events
func (eb *EventBuffer) GetClass3Count() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.class3.Len()
}

// ClearClass1 clears Class 1 events
func (eb *EventBuffer) ClearClass1() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.class1 = list.New()
}

// ClearClass2 clears Class 2 events
func (eb *EventBuffer) ClearClass2() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.class2 = list.New()
}

// ClearClass3 clears Class 3 events
func (eb *EventBuffer) ClearClass3() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.class3 = list.New()
}

// HasEvents returns true if any events are buffered
func (eb *EventBuffer) HasEvents() bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.class1.Len() > 0 || eb.class2.Len() > 0 || eb.class3.Len() > 0
}
