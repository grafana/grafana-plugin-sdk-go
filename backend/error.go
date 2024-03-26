package backend

type Error interface {
	error
	Source() ErrorSource
}

type pluginError struct {
	error
	source ErrorSource
}

func (e *pluginError) Source() ErrorSource {
	return e.source
}

func ErrorWithSource(err error, source ErrorSource) Error {
	return &pluginError{
		error:  err,
		source: source,
	}
}
