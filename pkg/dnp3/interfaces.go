package dnp3

import (
	//"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	//"github.com/lihongjie0209/dnp3-go/pkg/types"
)

// This file contains all interfaces and types that are used by both
// dnp3 package and master/outstation packages to avoid circular imports

// Re-export for convenience
type ClassField = app.ClassField

const (
	Class0   = app.Class0
	Class1   = app.Class1
	Class2   = app.Class2
	Class3   = app.Class3
	ClassAll = app.ClassAll
)
