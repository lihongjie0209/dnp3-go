package app

import (
	"github.com/lihongjie0209/dnp3-go/pkg/types"
	"time"
)

// Request builders - convenient functions for creating common DNP3 requests

// BuildReadRequest creates a read request APDU
func BuildReadRequest(seq uint8, objects []byte) *APDU {
	return NewRequestAPDU(FuncRead, seq, objects)
}

// BuildWriteRequest creates a write request APDU
func BuildWriteRequest(seq uint8, objects []byte) *APDU {
	return NewRequestAPDU(FuncWrite, seq, objects)
}

// BuildSelectRequest creates a select request APDU
func BuildSelectRequest(seq uint8, objects []byte) *APDU {
	return NewRequestAPDU(FuncSelect, seq, objects)
}

// BuildOperateRequest creates an operate request APDU
func BuildOperateRequest(seq uint8, objects []byte) *APDU {
	return NewRequestAPDU(FuncOperate, seq, objects)
}

// BuildDirectOperateRequest creates a direct operate request APDU
func BuildDirectOperateRequest(seq uint8, objects []byte) *APDU {
	return NewRequestAPDU(FuncDirectOperate, seq, objects)
}

// BuildConfirmRequest creates a confirm request APDU
func BuildConfirmRequest(seq uint8) *APDU {
	return NewRequestAPDU(FuncConfirm, seq, nil)
}

// BuildEnableUnsolicitedRequest creates an enable unsolicited request
func BuildEnableUnsolicitedRequest(seq uint8, classes ...ClassField) *APDU {
	objects := BuildEnableUnsolicited(classes...)
	return NewRequestAPDU(FuncEnableUnsolicited, seq, objects)
}

// BuildDisableUnsolicitedRequest creates a disable unsolicited request
func BuildDisableUnsolicitedRequest(seq uint8, classes ...ClassField) *APDU {
	objects := BuildDisableUnsolicited(classes...)
	return NewRequestAPDU(FuncDisableUnsolicited, seq, objects)
}

// BuildIntegrityPollRequest creates a Class 0 integrity poll request
func BuildIntegrityPollRequest(seq uint8) *APDU {
	objects := BuildIntegrityPoll()
	return BuildReadRequest(seq, objects)
}

// BuildEventPollRequest creates a Class 1,2,3 event poll request
func BuildEventPollRequest(seq uint8) *APDU {
	objects := BuildEventPoll()
	return BuildReadRequest(seq, objects)
}

// BuildTimeSyncRequest creates a time synchronization write request
func BuildTimeSyncRequest(seq uint8, t time.Time) *APDU {
	objects := BuildTimeSync(t)
	return BuildWriteRequest(seq, objects)
}

// BuildTimeSyncNowRequest creates a time sync request with current time
func BuildTimeSyncNowRequest(seq uint8) *APDU {
	return BuildTimeSyncRequest(seq, time.Now())
}

// BuildColdRestartRequest creates a cold restart request
func BuildColdRestartRequest(seq uint8) *APDU {
	return NewRequestAPDU(FuncColdRestart, seq, nil)
}

// BuildWarmRestartRequest creates a warm restart request
func BuildWarmRestartRequest(seq uint8) *APDU {
	return NewRequestAPDU(FuncWarmRestart, seq, nil)
}

// BuildSelectOperateRequest creates paired SELECT and OPERATE requests for CROB
func BuildSelectOperateRequest(startSeq uint8, index uint16, crob CROB) (*APDU, *APDU) {
	objects := BuildCROBRequest(index, crob)
	selectAPDU := BuildSelectRequest(startSeq, objects)
	operateAPDU := BuildOperateRequest(startSeq+1, objects)
	return selectAPDU, operateAPDU
}

// BuildDirectOperateCROBRequest creates a direct operate request for CROB
func BuildDirectOperateCROBRequest(seq uint8, index uint16, crob CROB) *APDU {
	objects := BuildCROBRequest(index, crob)
	return BuildDirectOperateRequest(seq, objects)
}

// Response builders - convenient functions for creating common DNP3 responses

// BuildEmptyResponse creates an empty response with just IIN
func BuildEmptyResponse(seq uint8, iin types.IIN) *APDU {
	return NewResponseAPDU(seq, iin, nil)
}

// BuildErrorResponse creates a response with error IIN flag
func BuildErrorResponse(seq uint8, errorFlag uint8) *APDU {
	var iin types.IIN
	iin.IIN2 = errorFlag
	return BuildEmptyResponse(seq, iin)
}

// BuildFunctionNotSupportedResponse creates a function not supported error response
func BuildFunctionNotSupportedResponse(seq uint8) *APDU {
	return BuildErrorResponse(seq, types.IIN2NoFuncCodeSupport)
}

// BuildObjectUnknownResponse creates an object unknown error response
func BuildObjectUnknownResponse(seq uint8) *APDU {
	return BuildErrorResponse(seq, types.IIN2ObjectUnknown)
}

// BuildParameterErrorResponse creates a parameter error response
func BuildParameterErrorResponse(seq uint8) *APDU {
	return BuildErrorResponse(seq, types.IIN2ParameterError)
}

// BuildDataResponse creates a response with data objects
func BuildDataResponse(seq uint8, iin types.IIN, objects []byte) *APDU {
	return NewResponseAPDU(seq, iin, objects)
}

// BuildUnsolicitedResponse creates an unsolicited response
func BuildUnsolicitedResponse(seq uint8, iin types.IIN, objects []byte) *APDU {
	return NewUnsolicitedResponseAPDU(seq, iin, objects)
}

// Sequence number management helpers

// SequenceCounter manages sequence numbers for requests
type SequenceCounter struct {
	current uint8
}

// NewSequenceCounter creates a new sequence counter starting at 0
func NewSequenceCounter() *SequenceCounter {
	return &SequenceCounter{current: 0}
}

// Next returns the next sequence number and increments the counter
func (s *SequenceCounter) Next() uint8 {
	seq := s.current
	s.current = (s.current + 1) & AppCtrlSeqMask // Wrap at 15
	return seq
}

// Current returns the current sequence number without incrementing
func (s *SequenceCounter) Current() uint8 {
	return s.current
}

// Reset resets the sequence counter to 0
func (s *SequenceCounter) Reset() {
	s.current = 0
}

// SetSequence sets the sequence to a specific value
func (s *SequenceCounter) SetSequence(seq uint8) {
	s.current = seq & AppCtrlSeqMask
}

// UnsolicitedSequenceCounter manages sequence for unsolicited responses
type UnsolicitedSequenceCounter struct {
	current uint8
}

// NewUnsolicitedSequenceCounter creates a new unsolicited sequence counter
func NewUnsolicitedSequenceCounter() *UnsolicitedSequenceCounter {
	return &UnsolicitedSequenceCounter{current: 0}
}

// Next returns the next sequence number for unsolicited
func (u *UnsolicitedSequenceCounter) Next() uint8 {
	seq := u.current
	u.current = (u.current + 1) & AppCtrlSeqMask
	return seq
}

// Current returns current unsolicited sequence
func (u *UnsolicitedSequenceCounter) Current() uint8 {
	return u.current
}

// Reset resets the unsolicited sequence counter
func (u *UnsolicitedSequenceCounter) Reset() {
	u.current = 0
}

// IIN Helper functions

// NewIIN creates a new IIN with no flags set
func NewIIN() types.IIN {
	return types.IIN{IIN1: 0, IIN2: 0}
}

// NewIINWithFlags creates an IIN with specific flags
func NewIINWithFlags(iin1, iin2 uint8) types.IIN {
	return types.IIN{IIN1: iin1, IIN2: iin2}
}

// NewIINWithEvents creates an IIN with event flags set
func NewIINWithEvents(class1, class2, class3 bool) types.IIN {
	var iin types.IIN
	if class1 {
		iin.IIN1 |= types.IIN1Class1Events
	}
	if class2 {
		iin.IIN1 |= types.IIN1Class2Events
	}
	if class3 {
		iin.IIN1 |= types.IIN1Class3Events
	}
	return iin
}

// NewIINNeedTime creates an IIN with NEED_TIME flag set
func NewIINNeedTime() types.IIN {
	return types.IIN{IIN1: types.IIN1NeedTime, IIN2: 0}
}

// NewIINDeviceRestart creates an IIN with DEVICE_RESTART flag set
func NewIINDeviceRestart() types.IIN {
	return types.IIN{IIN1: types.IIN1DeviceRestart, IIN2: 0}
}

// SetIINFlag sets a specific IIN flag
func SetIINFlag(iin *types.IIN, flag uint8, isIIN1 bool) {
	if isIIN1 {
		iin.IIN1 |= flag
	} else {
		iin.IIN2 |= flag
	}
}

// ClearIINFlag clears a specific IIN flag
func ClearIINFlag(iin *types.IIN, flag uint8, isIIN1 bool) {
	if isIIN1 {
		iin.IIN1 &^= flag
	} else {
		iin.IIN2 &^= flag
	}
}

// HasIINFlag checks if a specific IIN flag is set
func HasIINFlag(iin types.IIN, flag uint8, isIIN1 bool) bool {
	if isIIN1 {
		return (iin.IIN1 & flag) != 0
	}
	return (iin.IIN2 & flag) != 0
}

// IsIINError returns true if any error flag is set in IIN2
func IsIINError(iin types.IIN) bool {
	return iin.IIN2 != 0
}

// Control byte helpers

// BuildControlByte builds a control byte from components
func BuildControlByte(fir, fin, con, uns bool, seq uint8) uint8 {
	var ctrl uint8 = seq & AppCtrlSeqMask

	if fir {
		ctrl |= AppCtrlFIR
	}
	if fin {
		ctrl |= AppCtrlFIN
	}
	if con {
		ctrl |= AppCtrlCON
	}
	if uns {
		ctrl |= AppCtrlUNS
	}

	return ctrl
}

// ParseControlByte parses a control byte into components
func ParseControlByte(ctrl uint8) (fir, fin, con, uns bool, seq uint8) {
	fir = (ctrl & AppCtrlFIR) != 0
	fin = (ctrl & AppCtrlFIN) != 0
	con = (ctrl & AppCtrlCON) != 0
	uns = (ctrl & AppCtrlUNS) != 0
	seq = ctrl & AppCtrlSeqMask
	return
}
