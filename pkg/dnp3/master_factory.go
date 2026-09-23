package dnp3

import (
	"errors"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/internal/logger"
	"github.com/lihongjie0209/dnp3-go/pkg/master"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// newMaster creates a new master instance
func newMaster(config MasterConfig, callbacks MasterCallbacks, ch *channel.Channel, log logger.Logger) (Master, error) {
	// Convert dnp3 config to master config
	masterConfig := master.MasterConfig{
		ID:                    config.ID,
		LocalAddress:          config.LocalAddress,
		RemoteAddress:         config.RemoteAddress,
		ResponseTimeout:       config.ResponseTimeout,
		TaskRetryPeriod:       config.TaskRetryPeriod,
		TaskStartTimeout:      config.TaskStartTimeout,
		DisableUnsolOnStartup: config.DisableUnsolOnStartup,
		IgnoreRestartIIN:      config.IgnoreRestartIIN,
		UnsolClassMask:        config.UnsolClassMask,
		StartupIntegrityScan:  config.StartupIntegrityScan,
		IntegrityPeriod:       config.IntegrityPeriod,
		MaxRxFragSize:         config.MaxRxFragSize,
		MaxTxFragSize:         config.MaxTxFragSize,
	}

	wrappedCallbacks := &masterCallbacksWrapper{callbacks: callbacks}
	internalMaster, err := master.New(masterConfig, wrappedCallbacks, ch, log)
	if err != nil {
		return nil, err
	}

	return &masterWrapper{internal: internalMaster}, nil
}

// masterWrapper wraps internal master to implement public Master interface
type masterWrapper struct {
	internal interface {
		Enable() error
		Disable() error
		Shutdown() error
		AddIntegrityScan(period time.Duration) (master.ScanHandle, error)
		AddClassScan(classes app.ClassField, period time.Duration) (master.ScanHandle, error)
		AddRangeScan(objGroup, variation uint8, start, stop uint16, period time.Duration) (master.ScanHandle, error)
		ScanIntegrity() error
		ScanClasses(classes app.ClassField) error
		ScanRange(objGroup, variation uint8, start, stop uint16) error
		SelectAndOperate(commands []types.Command) ([]types.CommandStatus, error)
		DirectOperate(commands []types.Command) ([]types.CommandStatus, error)
	}
}

func (m *masterWrapper) Enable() error {
	return m.internal.Enable()
}

func (m *masterWrapper) Disable() error {
	return m.internal.Disable()
}

func (m *masterWrapper) Shutdown() error {
	return m.internal.Shutdown()
}

func (m *masterWrapper) SetConfig(config MasterConfig) error {
	// SetConfig is not supported - config must be set at creation time
	return errors.New("SetConfig not supported - config is immutable after creation")
}

func (m *masterWrapper) AddIntegrityScan(period time.Duration) (ScanHandle, error) {
	return m.internal.AddIntegrityScan(period)
}

func (m *masterWrapper) AddClassScan(classes app.ClassField, period time.Duration) (ScanHandle, error) {
	return m.internal.AddClassScan(classes, period)
}

func (m *masterWrapper) AddRangeScan(objGroup, variation uint8, start, stop uint16, period time.Duration) (ScanHandle, error) {
	return m.internal.AddRangeScan(objGroup, variation, start, stop, period)
}

func (m *masterWrapper) ScanIntegrity() error {
	return m.internal.ScanIntegrity()
}

func (m *masterWrapper) ScanClasses(classes app.ClassField) error {
	return m.internal.ScanClasses(classes)
}

func (m *masterWrapper) ScanRange(objGroup, variation uint8, start, stop uint16) error {
	return m.internal.ScanRange(objGroup, variation, start, stop)
}

func (m *masterWrapper) SelectAndOperate(commands []types.Command) ([]types.CommandStatus, error) {
	return m.internal.SelectAndOperate(commands)
}

func (m *masterWrapper) DirectOperate(commands []types.Command) ([]types.CommandStatus, error) {
	return m.internal.DirectOperate(commands)
}

// masterCallbacksWrapper wraps dnp3.MasterCallbacks to master.MasterCallbacks
type masterCallbacksWrapper struct {
	callbacks MasterCallbacks
}

func (w *masterCallbacksWrapper) OnBeginFragment(info master.ResponseInfo) {
	w.callbacks.OnBeginFragment(ResponseInfo{
		Unsolicited: info.Unsolicited,
		FIR:         info.FIR,
		FIN:         info.FIN,
	})
}

func (w *masterCallbacksWrapper) OnEndFragment(info master.ResponseInfo) {
	w.callbacks.OnEndFragment(ResponseInfo{
		Unsolicited: info.Unsolicited,
		FIR:         info.FIR,
		FIN:         info.FIN,
	})
}

func (w *masterCallbacksWrapper) ProcessBinary(info master.HeaderInfo, values []types.IndexedBinary) {
	w.callbacks.ProcessBinary(HeaderInfo{
		Group:     info.Group,
		Variation: info.Variation,
		Qualifier: info.Qualifier,
		IsEvent:   info.IsEvent,
	}, values)
}

func (w *masterCallbacksWrapper) ProcessDoubleBitBinary(info master.HeaderInfo, values []types.IndexedDoubleBitBinary) {
	w.callbacks.ProcessDoubleBitBinary(HeaderInfo{
		Group:     info.Group,
		Variation: info.Variation,
		Qualifier: info.Qualifier,
		IsEvent:   info.IsEvent,
	}, values)
}

func (w *masterCallbacksWrapper) ProcessAnalog(info master.HeaderInfo, values []types.IndexedAnalog) {
	w.callbacks.ProcessAnalog(HeaderInfo{
		Group:     info.Group,
		Variation: info.Variation,
		Qualifier: info.Qualifier,
		IsEvent:   info.IsEvent,
	}, values)
}

func (w *masterCallbacksWrapper) ProcessCounter(info master.HeaderInfo, values []types.IndexedCounter) {
	w.callbacks.ProcessCounter(HeaderInfo{
		Group:     info.Group,
		Variation: info.Variation,
		Qualifier: info.Qualifier,
		IsEvent:   info.IsEvent,
	}, values)
}

func (w *masterCallbacksWrapper) ProcessFrozenCounter(info master.HeaderInfo, values []types.IndexedFrozenCounter) {
	w.callbacks.ProcessFrozenCounter(HeaderInfo{
		Group:     info.Group,
		Variation: info.Variation,
		Qualifier: info.Qualifier,
		IsEvent:   info.IsEvent,
	}, values)
}

func (w *masterCallbacksWrapper) ProcessBinaryOutputStatus(info master.HeaderInfo, values []types.IndexedBinaryOutputStatus) {
	w.callbacks.ProcessBinaryOutputStatus(HeaderInfo{
		Group:     info.Group,
		Variation: info.Variation,
		Qualifier: info.Qualifier,
		IsEvent:   info.IsEvent,
	}, values)
}

func (w *masterCallbacksWrapper) ProcessAnalogOutputStatus(info master.HeaderInfo, values []types.IndexedAnalogOutputStatus) {
	w.callbacks.ProcessAnalogOutputStatus(HeaderInfo{
		Group:     info.Group,
		Variation: info.Variation,
		Qualifier: info.Qualifier,
		IsEvent:   info.IsEvent,
	}, values)
}

func (w *masterCallbacksWrapper) OnReceiveIIN(iin types.IIN) {
	w.callbacks.OnReceiveIIN(iin)
}

func (w *masterCallbacksWrapper) OnTaskStart(taskType master.TaskType, id int) {
	w.callbacks.OnTaskStart(TaskType(taskType), id)
}

func (w *masterCallbacksWrapper) OnTaskComplete(taskType master.TaskType, id int, result master.TaskResult) {
	w.callbacks.OnTaskComplete(TaskType(taskType), id, TaskResult(result))
}

func (w *masterCallbacksWrapper) GetTime() time.Time {
	return w.callbacks.GetTime()
}
