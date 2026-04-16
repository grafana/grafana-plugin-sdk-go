package v0alpha1

type QueryExamples struct {
	Examples []QueryExample `json:"examples"`
}

type QueryExample struct {
	// The example display name (shows up in swagger)
	Name string `json:"name"`

	// Query type matches the query type in the
	QueryType string `json:"queryType"`

	// Optionally explain why the example is interesting
	Description string `json:"description,omitempty"`

	// An example value saved that can be saved in a dashboard
	SaveModel Unstructured `json:"saveModel"`
}
