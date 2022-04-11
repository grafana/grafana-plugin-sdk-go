package sdata

import "list"

#FieldType: "time" | "number" | "string"
#FieldTypeInfoFrame: "time.Time" | "float64" | "string"  | "int64"
#FieldTypeInfo: {
	frame: #FieldTypeInfoFrame
}
#FieldSchema: {
	name: string
	type: #FieldType
	typeInfo: #FieldTypeInfo
	labels?: {
		[string]: string
	}
}
#FieldValue: string | number
#FieldValues: [#FieldValue, ...]
#FieldTypeInfoMap: {
	"time.Time": int
	float64: number 
	int64: int
	"string": string
}

#FrameType: "timeseries-many" | "timeseries-wide" | "timeseries-long"
#FrameMeta: {
	type: #FrameType
}
#FrameSchema: {
	name?: string
	fields: [#FieldSchema, ...]
	meta: #FrameMeta
}
#FrameData: {
	values: [#FieldValues, ...]
	#expectedLength: [ 
		for v in values { 
			v & list.MinItems(len(values[0])) & list.MaxItems(len(values[0])) 
		}
	]
}
#Frame: {
	schema: #FrameSchema
	data: #FrameData
	#matchingTypes: [
		for i, fv in data.values { 
			let typeInfo = schema.fields[i].typeInfo.frame
			let type = #FieldTypeInfoMap[typeInfo]
			[type, ...] & fv
		}
	]
}