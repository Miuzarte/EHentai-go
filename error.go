package EHentai

import (
	"fmt"
)

func wrapErr(err error, detail any) error {
	if err == nil {
		return nil
	}
	return &Error{raw: err, detail: detail}
}

func UnwrapErr(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return &Error{raw: err}
}

// Error 附带了更详细的信息
//
// ehErr := EHentai.UnwarpErr(err)
//
// isXxx := ehErr.Is(ErrXXX)
type Error struct {
	raw    error
	detail any
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.detail != nil {
		return fmt.Sprintf("%s: %v", e.raw.Error(), e.detail)
	}
	return e.raw.Error()
}

func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	return e.raw == target
}
