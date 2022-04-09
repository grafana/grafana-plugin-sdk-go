package wide 

import timeseries "github.com/grafana/grafana-plugin-sdk-go/experimental/sdata/timeseries"

#TimeField: timeseries.#FieldSchema & {
	type: "time",
	typeInfo: {
		frame: "time.Time"
	}
}

#NumberField: timeseries.#FieldSchema & {
	type: "number",
	typeInfo: {
		frame: "int64" | "float64"
	}
}

#WideFrame: timeseries.#Frame & {
	schema: timeseries.#FrameSchema & {
		meta: {
			type: "timeseries-wide"
		}
		fields: [#TimeField, ...#NumberField]
	}
	data: timeseries.#FrameData & {
		values: [[number, ...], ...]
	}
}

frames: [#WideFrame, ...]