package v0alpha1

type QueryExamples struct {
	Examples []QueryExample `json:"examples"`
}

type QueryExample struct {
	// The example display name (shows up in swagger)
	Name string `json:"name"`

	// Query type matches the query type from QueryTypeDefinitions (query.types.json)
	// The examples are then added to the swagger docs for this plugin
	QueryType string `json:"queryType"`

	// Optionally explain why the example is interesting
	Description string `json:"description,omitempty"`

	// An example value saved that can be saved in a dashboard
	SaveModel Unstructured `json:"saveModel"`
}
