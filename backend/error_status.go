package backend

type ErrorStatus int64

const (
	Undefined ErrorStatus = iota
	Timeout
	Unauthorized
	ConnectionError
)
