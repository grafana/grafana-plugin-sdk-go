package data

import (
	"encoding/csv"
	"fmt"
	"io"
)

type FrameToCSVOptions struct {
	ShowNames bool
	ShowTypes bool
}

func FrameToCSV(frame *Frame, writer io.Writer, opts FrameToCSVOptions) error {
	w := csv.NewWriter(writer)
	width := len(frame.Fields)
	rows := frame.Rows()
	line := make([]string, width)
	if opts.ShowNames {
		for i, f := range frame.Fields {
			line[i] = f.Name
		}
		if err := w.Write(line); err != nil {
			return err
		}
	}
	if opts.ShowTypes {
		for i, f := range frame.Fields {
			line[i] = f.Type().ItemTypeString()
		}
		if err := w.Write(line); err != nil {
			return err
		}
	}

	for row := 0; row < rows; row++ {
		for i, f := range frame.Fields {
			str := ""
			val, ok := f.ConcreteAt(row)
			if val != nil && ok {
				str = fmt.Sprintf("%v", val)
			}
			line[i] = str
		}
		if err := w.Write(line); err != nil {
			return err
		}
	}

	// Fulsh values and write
	w.Flush()
	return w.Error()
}
