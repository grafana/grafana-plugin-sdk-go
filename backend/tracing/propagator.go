package tracing

type PropagatorFormat string

const (
	PropagatorFormatJaeger PropagatorFormat = "jaeger"
	PropagatorFormatW3C    PropagatorFormat = "w3c"
)
