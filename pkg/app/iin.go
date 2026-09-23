package app

import "github.com/lihongjie0209/dnp3-go/pkg/types"

// Re-export IIN type from types package for convenience
type IIN = types.IIN

// Application layer IIN constants (re-export from types)
const (
	IIN1AllStations         = types.IIN1AllStations
	IIN1Class1Events        = types.IIN1Class1Events
	IIN1Class2Events        = types.IIN1Class2Events
	IIN1Class3Events        = types.IIN1Class3Events
	IIN1NeedTime            = types.IIN1NeedTime
	IIN1LocalControl        = types.IIN1LocalControl
	IIN1DeviceTrouble       = types.IIN1DeviceTrouble
	IIN1DeviceRestart       = types.IIN1DeviceRestart
	IIN2NoFuncCodeSupport   = types.IIN2NoFuncCodeSupport
	IIN2ObjectUnknown       = types.IIN2ObjectUnknown
	IIN2ParameterError      = types.IIN2ParameterError
	IIN2EventBufferOverflow = types.IIN2EventBufferOverflow
	IIN2AlreadyExecuting    = types.IIN2AlreadyExecuting
	IIN2ConfigCorrupt       = types.IIN2ConfigCorrupt
)
