package data

import (
	"errors"
	"reflect"
	"strings"
)

// Errors returned by the Marshal function
var (
	ErrorNotSlice      = errors.New("the data provided is not a slice")
	ErrorNotCollection = errors.New("slice element is not a struct or a map")
	ErrorChildNotfound = errors.New("referenced child element not found in tree")
)

// MarshalField is a field that is requested and turned into a field into a data.Frame
type MarshalField struct {
	Name  string
	Alias string
}

// in this tree datastructure, we will never refer to siblings of nodes.
// it will only be traversed through a known depth
type node struct {
	children map[string]*node
	val      reflect.Value
	t        reflect.Type
}

func newNode(val reflect.Value, t reflect.Type) *node {
	return &node{
		children: make(map[string]*node),
		val:      val,
		t:        t,
	}
}

// addChild sets or overwrites the child at "name" with "child"
func (n *node) addChild(name string, child *node) {
	if n.children == nil {
		n.children = make(map[string]*node)
	}

	n.children[name] = child
}

func (n *node) child(s string) (*node, error) {
	fields := strings.Split(s, ".")

	fieldNode := n
	for _, v := range fields {
		n, ok := fieldNode.children[v]
		if !ok {
			return nil, ErrorChildNotfound
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

func nodeToFields(node []*node, mf []MarshalField) (Fields, error) {
	fields := make(Fields, len(mf))
	for i, field := range mf {
		v := []interface{}{}
		for _, n := range node {
			n, err := n.child(field.Name)
			if err != nil {
				return nil, err
			}

			v = append(v, n.val.Interface())
		}

		name := field.Alias
		if name == "" {
			name = field.Name
		}

		fields[i] = NewField(name, nil, v)
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

// Marshal turns `v` into a list of data.Frames.
// The list of fields can contain periods to refer to sub-structs and map keys.
// If `v` is not a slice, then ErrorNotSlice is returned.
func Marshal(name string, fields []MarshalField, v interface{}) (*Frame, error) {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Slice {
		return nil, ErrorNotSlice
	}

	val := reflect.ValueOf(v)

	nodes := make([]*node, val.Len())
	for i := 0; i < val.Len(); i++ {
		t, err := newTree(val.Index(i))
		if err != nil {
			return nil, err
		}
		nodes[i] = t
	}

	nodeFields, err := nodeToFields(nodes, fields)
	if err != nil {
		return nil, err
	}

	return NewFrame(name, nodeFields...), nil
}
