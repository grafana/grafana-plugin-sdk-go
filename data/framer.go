package data

// Framer is simply an object that can be converted to Grafana data frames.
// This interface allows us to interact with types that represent datasource objects
// Without having to convert them to dataframes first
type Framer interface {
	Frames() Frames
}
