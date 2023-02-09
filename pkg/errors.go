package pkg

import "errors"

var (
	ErrMQTTCodeProtocolError = errors.New(`mqtt protocl code error`)
	ErrSwitchType            = errors.New(`swi`)
)
