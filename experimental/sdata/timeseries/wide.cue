package timeseries 

import sdata "github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"

#TimeField: sdata.#FieldSchema & {
	type: "time",
	typeInfo: {
		frame: "time.Time"
	}
}

#NumberField: sdata.#FieldSchema & {
	type: "number",
	typeInfo: {
		frame: "int64" | "float64"
	}
}

#WideFrame: sdata.#Frame & {
	schema: sdata.#FrameSchema & {
		meta: {
			type: "timeseries-wide"
		}
		fields: [#TimeField, ...#NumberField]
	}
	data: sdata.#FrameData & {
		values: [[number, ...], ...]
	}
}

frames: [#WideFrame, ...]