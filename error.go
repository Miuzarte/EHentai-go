package EHentai

import "fmt"

func wrapErr(err error, detail any) error {
	if err == nil {
		return nil
	}
	return &Error{raw: err, detail: detail}
}

type Error struct {
	raw    error
	detail any
}

func (e *Error) Error() string {
	if e.detail != nil {
		return fmt.Sprintf("%s: %v", e.raw.Error(), e.detail)
	}
	return e.raw.Error()
}

func (e *Error) Unwrap() error {
	return e.raw
}

func (e *Error) Is(target error) bool {
	return e.raw == target
}
