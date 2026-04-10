//go:build darwin

package auth

/*
#cgo LDFLAGS: -framework LocalAuthentication -framework Foundation
#include <stdlib.h>
#include "touchid.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

var ErrNotAvailable = errors.New("Touch ID not available on this device")

// Authenticate shows a Touch ID prompt with the given reason.
// Returns true on success, false on user cancel, ErrNotAvailable if Touch ID is not supported.
func Authenticate(reason string) (bool, error) {
	cr := C.CString(reason)
	defer C.free(unsafe.Pointer(cr))

	r := C.authenticateTouchID(cr)
	switch r {
	case 1:
		return true, nil
	case -1:
		return false, ErrNotAvailable
	default:
		return false, nil
	}
}
