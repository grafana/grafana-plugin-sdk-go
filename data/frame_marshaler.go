package data

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Errors returned by the Marshal function
var (
	ErrorNotSlice         = errors.New("the data provided is not a slice")
	ErrorNotCollection    = errors.New("slice element is not a struct or a map")
	ErrorChildNotfound    = errors.New("referenced child element not found in tree")
	ErrorUnrecognizedType = errors.New("unrecognized type")
	ErrorNoData           = errors.New("no data in slice")
)

// MarshalField is a field that is requested and turned into a field into a data.Frame
type MarshalField struct {
	Name  string
	Alias string
}

type nodes []*node

func (n nodes) peek(path string) (*node, error) {
	if len(n) == 0 {
		return nil, nil
	}

	return n[0].child(path)
}

func (n nodes) get(path string) (nodes, error) {
	res := make(nodes, len(n))
	for i, node := range n {
		child, err := node.child(path)
		if err != nil {
			return nil, err
		}
		res[i] = child
	}

	return res, nil
}

// in this tree datastructure, we will never refer to siblings of nodes.
// it will only be traversed through a known depth
type node struct {
	Children map[string]*node
	Val      reflect.Value
	T        reflect.Type
}

func newNode(val reflect.Value, t reflect.Type) *node {
	return &node{
		Children: make(map[string]*node),
		Val:      val,
		T:        t,
	}
}

// addChild sets or overwrites the child at "name" with "child"
func (n *node) addChild(name string, child *node) {
	if n.Children == nil {
		n.Children = make(map[string]*node)
	}

	n.Children[name] = child
}

func (n *node) child(s string) (*node, error) {
	fields := strings.Split(s, ".")

	fieldNode := n
	for i, v := range fields {
		n, ok := fieldNode.Children[v]
		if !ok {
			return nil, fmt.Errorf("[%d] %s: %w", i, v, ErrorChildNotfound)
		}
		fieldNode = n
	}

	return fieldNode, nil
}

func newTreeStruct(v reflect.Value) *node {
	t := v.Type()
	root := newNode(v, t)

	v.FieldByNameFunc(func(name string) bool {
		v := v.FieldByName(name)
		field, ok := t.FieldByName(name)
		if !ok {
			return false
		}

		node := newNode(v, field.Type)

		if field.Type.Kind() == reflect.Struct || field.Type.Kind() == reflect.Map {
			tree, err := newTree(v)
			if err != nil {
				return false
			}
			node = tree
		}

		if f := field.Tag.Get("frame"); f != "" {
			name = f
		}

		root.addChild(name, node)
		return false
	})

	return root
}

func newTreeMap(v reflect.Value) *node {
	return nil
}

func newField(mf MarshalField, nodes nodes) (*Field, error) {
	peeked, err := nodes.peek(mf.Name)
	if err != nil {
		return nil, err
	}

	tf, ok := typeFuncs[peeked.Val.Kind()]
	if !ok {
		return nil, ErrorUnrecognizedType
	}

	n, err := nodes.get(mf.Name)
	if err != nil {
		return nil, err
	}

	name := mf.Alias
	if name == "" {
		name = mf.Name
	}

	return tf(name, peeked.T, n)
}

func nodeToFields(nodes []*node, mf []MarshalField) (Fields, error) {
	fields := make(Fields, len(mf))
	for i, field := range mf {
		f, err := newField(field, nodes)
		if err != nil {
			return nil, err
		}

		fields[i] = f
	}

	return fields, nil
}

// newTree creates a new tree from v
// v must either be a map or a struct.
func newTree(v reflect.Value) (*node, error) {
	t := v.Type()
	if t.Kind() == reflect.Struct {
		return newTreeStruct(v), nil
	}

	if t.Kind() == reflect.Map {
		return newTreeMap(v), nil
	}

	return nil, ErrorNotCollection
}

func newTreeList(v interface{}) (nodes, error) {
	val := reflect.ValueOf(v)

	nodes := make([]*node, val.Len())
	for i := 0; i < val.Len(); i++ {
		t, err := newTree(val.Index(i))
		if err != nil {
			return nil, err
		}
		nodes[i] = t
	}

	return nodes, nil
}

// Marshal turns `v` into a list of data.Frames.
// The list of fields can contain periods to refer to sub-structs and map keys.
// If `v` is not a slice, then ErrorNotSlice is returned.
func Marshal(name string, fields []MarshalField, v interface{}) (*Frame, error) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Slice {
		return nil, ErrorNotSlice
	}

	val := reflect.ValueOf(v)
	if val.Len() == 0 {
		return nil, ErrorNoData
	}

	nodes, err := newTreeList(v)
	if err != nil {
		return nil, err
	}

	nodeFields, err := nodeToFields(nodes, fields)
	if err != nil {
		return nil, err
	}

	return NewFrame(name, nodeFields...), nil
}

// MarshalFields creates a list of fields to provide to "Marshal"
// This function does not handle aliases. If you need aliases, you should create the `[]MarshalField` list yourself
func MarshalFields(fields ...string) []MarshalField {
	mf := make([]MarshalField, len(fields))
	for i, v := range fields {
		mf[i] = MarshalField{
			Name: v,
		}
	}

	return mf
}
