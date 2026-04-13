package patterns

import (
	"go/ast"

	"golang.org/x/tools/go/ast/inspector"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

type Context struct {
	Load      *load.Result
	Inspector *inspector.Inspector
	Report    *model.Report
}

type PendingKind string

const (
	PendingDecodeTarget PendingKind = "decode_target"
	PendingSecureKey    PendingKind = "secure_key"
)

type Pending struct {
	Kind         PendingKind
	PackagePath  string
	FunctionName string
	Node         ast.Node
	Reason       string
}

type Result struct {
	Report  model.Report
	Pending []Pending
}
