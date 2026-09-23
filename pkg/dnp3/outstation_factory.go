package dnp3

import (
	"errors"

	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/internal/logger"
	"github.com/lihongjie0209/dnp3-go/pkg/outstation"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// newOutstation creates a new outstation instance
func newOutstation(config OutstationConfig, callbacks OutstationCallbacks, ch *channel.Channel, log logger.Logger) (Outstation, error) {
	// Convert dnp3 config to outstation config
	outstationConfig := outstation.OutstationConfig{
		ID:                    config.ID,
		LocalAddress:          config.LocalAddress,
		RemoteAddress:         config.RemoteAddress,
		Database:              convertDatabaseConfig(config.Database),
		MaxBinaryEvents:       config.MaxBinaryEvents,
		MaxAnalogEvents:       config.MaxAnalogEvents,
		MaxCounterEvents:      config.MaxCounterEvents,
		MaxDoubleBitEvents:    config.MaxDoubleBitEvents,
		AllowUnsolicited:      config.AllowUnsolicited,
		UnsolConfirmTimeout:   config.UnsolConfirmTimeout,
		SelectTimeout:         config.SelectTimeout,
		MaxControlsPerRequest: config.MaxControlsPerRequest,
		LocalControl:          config.LocalControl,
		DeviceTrouble:         config.DeviceTrouble,
		MaxRxFragSize:         config.MaxRxFragSize,
		MaxTxFragSize:         config.MaxTxFragSize,
	}

	wrappedCallbacks := &outstationCallbacksWrapper{callbacks: callbacks}
	internalOutstation, err := outstation.New(outstationConfig, wrappedCallbacks, ch, log)
	if err != nil {
		return nil, err
	}

	return &outstationWrapper{internal: internalOutstation}, nil
}

func convertDatabaseConfig(config DatabaseConfig) outstation.DatabaseConfig {
	return outstation.DatabaseConfig{
		Binary:        convertBinaryConfigs(config.Binary),
		DoubleBit:     convertDoubleBitConfigs(config.DoubleBit),
		Analog:        convertAnalogConfigs(config.Analog),
		Counter:       convertCounterConfigs(config.Counter),
		FrozenCounter: convertFrozenCounterConfigs(config.FrozenCounter),
		BinaryOutput:  convertBinaryOutputConfigs(config.BinaryOutput),
		AnalogOutput:  convertAnalogOutputConfigs(config.AnalogOutput),
	}
}

func convertBinaryConfigs(configs []BinaryPointConfig) []outstation.BinaryPointConfig {
	result := make([]outstation.BinaryPointConfig, len(configs))
	for i, c := range configs {
		result[i] = outstation.BinaryPointConfig{
			StaticVariation: c.StaticVariation,
			EventVariation:  c.EventVariation,
			Class:           c.Class,
		}
	}
	return result
}

func convertDoubleBitConfigs(configs []DoubleBitBinaryPointConfig) []outstation.DoubleBitBinaryPointConfig {
	result := make([]outstation.DoubleBitBinaryPointConfig, len(configs))
	for i, c := range configs {
		result[i] = outstation.DoubleBitBinaryPointConfig{
			StaticVariation: c.StaticVariation,
			EventVariation:  c.EventVariation,
			Class:           c.Class,
		}
	}
	return result
}

func convertAnalogConfigs(configs []AnalogPointConfig) []outstation.AnalogPointConfig {
	result := make([]outstation.AnalogPointConfig, len(configs))
	for i, c := range configs {
		result[i] = outstation.AnalogPointConfig{
			StaticVariation: c.StaticVariation,
			EventVariation:  c.EventVariation,
			Class:           c.Class,
			Deadband:        c.Deadband,
		}
	}
	return result
}

func convertCounterConfigs(configs []CounterPointConfig) []outstation.CounterPointConfig {
	result := make([]outstation.CounterPointConfig, len(configs))
	for i, c := range configs {
		result[i] = outstation.CounterPointConfig{
			StaticVariation: c.StaticVariation,
			EventVariation:  c.EventVariation,
			Class:           c.Class,
			Deadband:        c.Deadband,
		}
	}
	return result
}

func convertFrozenCounterConfigs(configs []FrozenCounterPointConfig) []outstation.FrozenCounterPointConfig {
	result := make([]outstation.FrozenCounterPointConfig, len(configs))
	for i, c := range configs {
		result[i] = outstation.FrozenCounterPointConfig{
			StaticVariation: c.StaticVariation,
			EventVariation:  c.EventVariation,
			Class:           c.Class,
		}
	}
	return result
}

func convertBinaryOutputConfigs(configs []BinaryOutputStatusPointConfig) []outstation.BinaryOutputStatusPointConfig {
	result := make([]outstation.BinaryOutputStatusPointConfig, len(configs))
	for i, c := range configs {
		result[i] = outstation.BinaryOutputStatusPointConfig{
			StaticVariation: c.StaticVariation,
			EventVariation:  c.EventVariation,
			Class:           c.Class,
		}
	}
	return result
}

func convertAnalogOutputConfigs(configs []AnalogOutputStatusPointConfig) []outstation.AnalogOutputStatusPointConfig {
	result := make([]outstation.AnalogOutputStatusPointConfig, len(configs))
	for i, c := range configs {
		result[i] = outstation.AnalogOutputStatusPointConfig{
			StaticVariation: c.StaticVariation,
			EventVariation:  c.EventVariation,
			Class:           c.Class,
			Deadband:        c.Deadband,
		}
	}
	return result
}

// outstationCallbacksWrapper wraps dnp3.OutstationCallbacks to outstation.OutstationCallbacks
type outstationCallbacksWrapper struct {
	callbacks OutstationCallbacks
}

func (w *outstationCallbacksWrapper) Begin() {
	w.callbacks.Begin()
}

func (w *outstationCallbacksWrapper) End() {
	w.callbacks.End()
}

func (w *outstationCallbacksWrapper) SelectCROB(crob types.CROB, index uint16) types.CommandStatus {
	return w.callbacks.SelectCROB(crob, index)
}

func (w *outstationCallbacksWrapper) OperateCROB(crob types.CROB, index uint16, opType outstation.OperateType, handler outstation.UpdateHandler) types.CommandStatus {
	wrappedHandler := &updateHandlerWrapper{handler: handler}
	return w.callbacks.OperateCROB(crob, index, OperateType(opType), wrappedHandler)
}

func (w *outstationCallbacksWrapper) SelectAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16) types.CommandStatus {
	return w.callbacks.SelectAnalogOutputInt32(ao, index)
}

func (w *outstationCallbacksWrapper) OperateAnalogOutputInt32(ao types.AnalogOutputInt32, index uint16, opType outstation.OperateType, handler outstation.UpdateHandler) types.CommandStatus {
	wrappedHandler := &updateHandlerWrapper{handler: handler}
	return w.callbacks.OperateAnalogOutputInt32(ao, index, OperateType(opType), wrappedHandler)
}

func (w *outstationCallbacksWrapper) SelectAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16) types.CommandStatus {
	return w.callbacks.SelectAnalogOutputInt16(ao, index)
}

func (w *outstationCallbacksWrapper) OperateAnalogOutputInt16(ao types.AnalogOutputInt16, index uint16, opType outstation.OperateType, handler outstation.UpdateHandler) types.CommandStatus {
	wrappedHandler := &updateHandlerWrapper{handler: handler}
	return w.callbacks.OperateAnalogOutputInt16(ao, index, OperateType(opType), wrappedHandler)
}

func (w *outstationCallbacksWrapper) SelectAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16) types.CommandStatus {
	return w.callbacks.SelectAnalogOutputFloat32(ao, index)
}

func (w *outstationCallbacksWrapper) OperateAnalogOutputFloat32(ao types.AnalogOutputFloat32, index uint16, opType outstation.OperateType, handler outstation.UpdateHandler) types.CommandStatus {
	wrappedHandler := &updateHandlerWrapper{handler: handler}
	return w.callbacks.OperateAnalogOutputFloat32(ao, index, OperateType(opType), wrappedHandler)
}

func (w *outstationCallbacksWrapper) SelectAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16) types.CommandStatus {
	return w.callbacks.SelectAnalogOutputDouble64(ao, index)
}

func (w *outstationCallbacksWrapper) OperateAnalogOutputDouble64(ao types.AnalogOutputDouble64, index uint16, opType outstation.OperateType, handler outstation.UpdateHandler) types.CommandStatus {
	wrappedHandler := &updateHandlerWrapper{handler: handler}
	return w.callbacks.OperateAnalogOutputDouble64(ao, index, OperateType(opType), wrappedHandler)
}

func (w *outstationCallbacksWrapper) OnConfirmReceived(unsolicited bool, numClass1, numClass2, numClass3 uint) {
	w.callbacks.OnConfirmReceived(unsolicited, numClass1, numClass2, numClass3)
}

func (w *outstationCallbacksWrapper) OnUnsolicitedResponse(success bool, seq uint8) {
	w.callbacks.OnUnsolicitedResponse(success, seq)
}

func (w *outstationCallbacksWrapper) GetApplicationIIN() types.IIN {
	return w.callbacks.GetApplicationIIN()
}

// updateHandlerWrapper wraps UpdateHandler
type updateHandlerWrapper struct {
	handler outstation.UpdateHandler
}

func (w *updateHandlerWrapper) Update(meas interface{}, index uint16, mode EventMode) bool {
	return w.handler.Update(meas, index, outstation.EventMode(mode))
}

// NewUpdateBuilder creates a new update builder
func NewUpdateBuilder() *UpdateBuilder {
	return &UpdateBuilder{
		builder: outstation.NewUpdateBuilder(),
	}
}

// UpdateBuilder wraps the internal update builder
type UpdateBuilder struct {
	builder *outstation.UpdateBuilder
}

// UpdateBinary updates a binary point
func (b *UpdateBuilder) UpdateBinary(value types.Binary, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateBinary(value, index, outstation.EventMode(mode))
	return b
}

// UpdateAnalog updates an analog point
func (b *UpdateBuilder) UpdateAnalog(value types.Analog, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateAnalog(value, index, outstation.EventMode(mode))
	return b
}

// UpdateCounter updates a counter point
func (b *UpdateBuilder) UpdateCounter(value types.Counter, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateCounter(value, index, outstation.EventMode(mode))
	return b
}

// UpdateDoubleBitBinary updates a double-bit binary point
func (b *UpdateBuilder) UpdateDoubleBitBinary(value types.DoubleBitBinary, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateDoubleBitBinary(value, index, outstation.EventMode(mode))
	return b
}

// UpdateFrozenCounter updates a frozen counter point
func (b *UpdateBuilder) UpdateFrozenCounter(value types.FrozenCounter, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateFrozenCounter(value, index, outstation.EventMode(mode))
	return b
}

// UpdateBinaryOutputStatus updates a binary output status point
func (b *UpdateBuilder) UpdateBinaryOutputStatus(value types.BinaryOutputStatus, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateBinaryOutputStatus(value, index, outstation.EventMode(mode))
	return b
}

// UpdateAnalogOutputStatus updates an analog output status point
func (b *UpdateBuilder) UpdateAnalogOutputStatus(value types.AnalogOutputStatus, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateAnalogOutputStatus(value, index, outstation.EventMode(mode))
	return b
}

// Build builds the updates
func (b *UpdateBuilder) Build() *Updates {
	internalUpdates := b.builder.Build()
	return &Updates{Data: internalUpdates}
}

// outstationWrapper wraps internal outstation to implement public Outstation interface
type outstationWrapper struct {
	internal interface {
		Enable() error
		Disable() error
		Shutdown() error
		Apply(updates *outstation.Updates) error
		SetConfig(config outstation.OutstationConfig) error
	}
}

func (o *outstationWrapper) Enable() error {
	return o.internal.Enable()
}

func (o *outstationWrapper) Disable() error {
	return o.internal.Disable()
}

func (o *outstationWrapper) Shutdown() error {
	return o.internal.Shutdown()
}

func (o *outstationWrapper) Apply(updates *Updates) error {
	// Extract internal updates from wrapper
	internalUpdates, ok := updates.Data.(*outstation.Updates)
	if !ok {
		return errors.New("invalid updates type")
	}
	return o.internal.Apply(internalUpdates)
}

func (o *outstationWrapper) SetConfig(config OutstationConfig) error {
	outstationConfig := outstation.OutstationConfig{
		ID:                    config.ID,
		LocalAddress:          config.LocalAddress,
		RemoteAddress:         config.RemoteAddress,
		Database:              convertDatabaseConfig(config.Database),
		MaxBinaryEvents:       config.MaxBinaryEvents,
		MaxAnalogEvents:       config.MaxAnalogEvents,
		MaxCounterEvents:      config.MaxCounterEvents,
		MaxDoubleBitEvents:    config.MaxDoubleBitEvents,
		AllowUnsolicited:      config.AllowUnsolicited,
		UnsolConfirmTimeout:   config.UnsolConfirmTimeout,
		SelectTimeout:         config.SelectTimeout,
		MaxControlsPerRequest: config.MaxControlsPerRequest,
		LocalControl:          config.LocalControl,
		DeviceTrouble:         config.DeviceTrouble,
		MaxRxFragSize:         config.MaxRxFragSize,
		MaxTxFragSize:         config.MaxTxFragSize,
	}
	return o.internal.SetConfig(outstationConfig)
}
