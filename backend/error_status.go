package backend

type ErrorStatus int32

const (
	Undefined ErrorStatus = iota
	Timeout
	Unauthorized
	ConnectionError
)
