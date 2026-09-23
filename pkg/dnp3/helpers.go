package dnp3

import (
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// Command helpers for convenient command creation

// NewCROBCommand creates a new CROB command for a specific index
func NewCROBCommand(index uint16, crob types.CROB) types.Command {
	return types.Command{
		Index: index,
		Data:  crob,
	}
}

// NewLatchOnCommand creates a CROB command to latch output ON
func NewLatchOnCommand(index uint16) types.Command {
	return types.Command{
		Index: index,
		Data: types.CROB{
			OpType:    types.ControlCodeLatchOn,
			Count:     1,
			OnTimeMs:  0,
			OffTimeMs: 0,
		},
	}
}

// NewLatchOffCommand creates a CROB command to latch output OFF
func NewLatchOffCommand(index uint16) types.Command {
	return types.Command{
		Index: index,
		Data: types.CROB{
			OpType:    types.ControlCodeLatchOff,
			Count:     1,
			OnTimeMs:  0,
			OffTimeMs: 0,
		},
	}
}

// NewPulseOnCommand creates a CROB command to pulse output ON
func NewPulseOnCommand(index uint16, onTimeMs uint32) types.Command {
	return types.Command{
		Index: index,
		Data: types.CROB{
			OpType:    types.ControlCodePulseOn,
			Count:     1,
			OnTimeMs:  onTimeMs,
			OffTimeMs: 0,
		},
	}
}

// NewPulseOffCommand creates a CROB command to pulse output OFF
func NewPulseOffCommand(index uint16, offTimeMs uint32) types.Command {
	return types.Command{
		Index: index,
		Data: types.CROB{
			OpType:    types.ControlCodePulseOff,
			Count:     1,
			OnTimeMs:  0,
			OffTimeMs: offTimeMs,
		},
	}
}

// NewAnalogOutputInt32Command creates an analog output command with int32 value
func NewAnalogOutputInt32Command(index uint16, value int32) types.Command {
	return types.Command{
		Index: index,
		Data: types.AnalogOutputInt32{
			Value: value,
		},
	}
}

// NewAnalogOutputInt16Command creates an analog output command with int16 value
func NewAnalogOutputInt16Command(index uint16, value int16) types.Command {
	return types.Command{
		Index: index,
		Data: types.AnalogOutputInt16{
			Value: value,
		},
	}
}

// NewAnalogOutputFloat32Command creates an analog output command with float32 value
func NewAnalogOutputFloat32Command(index uint16, value float32) types.Command {
	return types.Command{
		Index: index,
		Data: types.AnalogOutputFloat32{
			Value: value,
		},
	}
}

// NewAnalogOutputDouble64Command creates an analog output command with float64 value
func NewAnalogOutputDouble64Command(index uint16, value float64) types.Command {
	return types.Command{
		Index: index,
		Data: types.AnalogOutputDouble64{
			Value: value,
		},
	}
}
