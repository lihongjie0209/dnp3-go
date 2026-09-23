package outstation

import (
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// UpdateBuilder builds atomic measurement updates
type UpdateBuilder struct {
	updates map[updateKey]updateValue
}

type updateKey struct {
	pointType MeasurementType
	index     uint16
}

type updateValue struct {
	measurement interface{}
	mode        EventMode
}

// NewUpdateBuilder creates a new update builder
func NewUpdateBuilder() *UpdateBuilder {
	return &UpdateBuilder{
		updates: make(map[updateKey]updateValue),
	}
}

// UpdateBinary updates a binary point
func (b *UpdateBuilder) UpdateBinary(value types.Binary, index uint16, mode EventMode) *UpdateBuilder {
	key := updateKey{pointType: MeasurementTypeBinary, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateDoubleBitBinary updates a double-bit binary point
func (b *UpdateBuilder) UpdateDoubleBitBinary(value types.DoubleBitBinary, index uint16, mode EventMode) *UpdateBuilder {
	key := updateKey{pointType: MeasurementTypeDoubleBitBinary, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateAnalog updates an analog point
func (b *UpdateBuilder) UpdateAnalog(value types.Analog, index uint16, mode EventMode) *UpdateBuilder {
	key := updateKey{pointType: MeasurementTypeAnalog, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateCounter updates a counter point
func (b *UpdateBuilder) UpdateCounter(value types.Counter, index uint16, mode EventMode) *UpdateBuilder {
	key := updateKey{pointType: MeasurementTypeCounter, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateFrozenCounter updates a frozen counter point
func (b *UpdateBuilder) UpdateFrozenCounter(value types.FrozenCounter, index uint16, mode EventMode) *UpdateBuilder {
	key := updateKey{pointType: MeasurementTypeFrozenCounter, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateBinaryOutputStatus updates a binary output status point
func (b *UpdateBuilder) UpdateBinaryOutputStatus(value types.BinaryOutputStatus, index uint16, mode EventMode) *UpdateBuilder {
	key := updateKey{pointType: MeasurementTypeBinaryOutputStatus, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateAnalogOutputStatus updates an analog output status point
func (b *UpdateBuilder) UpdateAnalogOutputStatus(value types.AnalogOutputStatus, index uint16, mode EventMode) *UpdateBuilder {
	key := updateKey{pointType: MeasurementTypeAnalogOutputStatus, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// Build builds the updates object
func (b *UpdateBuilder) Build() *Updates {
	return &Updates{
		Data: b.updates,
	}
}

// GetUpdates returns the updates map (for internal use)
func (b *UpdateBuilder) GetUpdates() map[updateKey]updateValue {
	return b.updates
}
