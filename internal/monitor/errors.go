package monitor

import "errors"

var (
	ErrUnknownKeyError              = errors.New("unknown key")
	ErrUnsupportedMetricMethodError = errors.New("unsupported metric method")
	ErrIntegerOutOfRange            = errors.New("integer out of range of uint64")
	ErrInvalidTimeError             = errors.New("invalid time")
	ErrUnknownValueType             = errors.New("unknown value type")
)
