package model

type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

type DecodeKind string

const (
	DecodeKindJSONUnmarshal   DecodeKind = "json_unmarshal"
	DecodeKindMapstructure    DecodeKind = "mapstructure_decode"
	DecodeKindSecureLiteral   DecodeKind = "secure_literal"
	DecodeKindSecureTemplate  DecodeKind = "secure_template"
	DecodeKindSecureBulkRange DecodeKind = "secure_bulk_range"
)

type SourceKind string

const (
	SourceKindDatasourceJSON   SourceKind = "datasource_json"
	SourceKindDatasourceSecure SourceKind = "datasource_secure_json"
	SourceKindQueryJSON        SourceKind = "query_json"
)

type Position struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type TargetRef struct {
	PackagePath string    `json:"packagePath"`
	TypeName    string    `json:"typeName"`
	Pointer     bool      `json:"pointer"`
	TypeString  string    `json:"typeString,omitempty"`
	Expr        *Position `json:"expr,omitempty"`
}

type Finding struct {
	Kind         DecodeKind `json:"kind"`
	Source       SourceKind `json:"source"`
	Position     Position   `json:"position"`
	FunctionName string     `json:"functionName,omitempty"`
	Target       *TargetRef `json:"target,omitempty"`
	Key          string     `json:"key,omitempty"`
	Pattern      string     `json:"pattern,omitempty"`
	Destination  string     `json:"destination,omitempty"`
	Confidence   Confidence `json:"confidence"`
	Notes        []string   `json:"notes,omitempty"`
}

type Warning struct {
	Position Position `json:"position"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
}

type Report struct {
	Findings []Finding `json:"findings"`
	Warnings []Warning `json:"warnings,omitempty"`
}
