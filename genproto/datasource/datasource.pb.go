// Code generated by protoc-gen-go. DO NOT EDIT.
// source: datasource.proto

package datasource

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type RowValue_Type int32

const (
	// Field type null.
	RowValue_TYPE_NULL RowValue_Type = 0
	// Field type double.
	RowValue_TYPE_DOUBLE RowValue_Type = 1
	// Field type int64.
	RowValue_TYPE_INT64 RowValue_Type = 2
	// Field type bool.
	RowValue_TYPE_BOOL RowValue_Type = 3
	// Field type string.
	RowValue_TYPE_STRING RowValue_Type = 4
	// Field type bytes.
	RowValue_TYPE_BYTES RowValue_Type = 5
)

var RowValue_Type_name = map[int32]string{
	0: "TYPE_NULL",
	1: "TYPE_DOUBLE",
	2: "TYPE_INT64",
	3: "TYPE_BOOL",
	4: "TYPE_STRING",
	5: "TYPE_BYTES",
}

var RowValue_Type_value = map[string]int32{
	"TYPE_NULL":   0,
	"TYPE_DOUBLE": 1,
	"TYPE_INT64":  2,
	"TYPE_BOOL":   3,
	"TYPE_STRING": 4,
	"TYPE_BYTES":  5,
}

func (x RowValue_Type) String() string {
	return proto.EnumName(RowValue_Type_name, int32(x))
}

func (RowValue_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{8, 0}
}

type DatasourceRequest struct {
	TimeRange            *TimeRange      `protobuf:"bytes,1,opt,name=timeRange,proto3" json:"timeRange,omitempty"`
	Datasource           *DatasourceInfo `protobuf:"bytes,2,opt,name=datasource,proto3" json:"datasource,omitempty"`
	Queries              []*Query        `protobuf:"bytes,3,rep,name=queries,proto3" json:"queries,omitempty"`
	RequestId            uint32          `protobuf:"varint,4,opt,name=requestId,proto3" json:"requestId,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *DatasourceRequest) Reset()         { *m = DatasourceRequest{} }
func (m *DatasourceRequest) String() string { return proto.CompactTextString(m) }
func (*DatasourceRequest) ProtoMessage()    {}
func (*DatasourceRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{0}
}

func (m *DatasourceRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DatasourceRequest.Unmarshal(m, b)
}
func (m *DatasourceRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DatasourceRequest.Marshal(b, m, deterministic)
}
func (m *DatasourceRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DatasourceRequest.Merge(m, src)
}
func (m *DatasourceRequest) XXX_Size() int {
	return xxx_messageInfo_DatasourceRequest.Size(m)
}
func (m *DatasourceRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DatasourceRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DatasourceRequest proto.InternalMessageInfo

func (m *DatasourceRequest) GetTimeRange() *TimeRange {
	if m != nil {
		return m.TimeRange
	}
	return nil
}

func (m *DatasourceRequest) GetDatasource() *DatasourceInfo {
	if m != nil {
		return m.Datasource
	}
	return nil
}

func (m *DatasourceRequest) GetQueries() []*Query {
	if m != nil {
		return m.Queries
	}
	return nil
}

func (m *DatasourceRequest) GetRequestId() uint32 {
	if m != nil {
		return m.RequestId
	}
	return 0
}

type Query struct {
	RefId                string   `protobuf:"bytes,1,opt,name=refId,proto3" json:"refId,omitempty"`
	MaxDataPoints        int64    `protobuf:"varint,2,opt,name=maxDataPoints,proto3" json:"maxDataPoints,omitempty"`
	IntervalMs           int64    `protobuf:"varint,3,opt,name=intervalMs,proto3" json:"intervalMs,omitempty"`
	ModelJson            string   `protobuf:"bytes,4,opt,name=modelJson,proto3" json:"modelJson,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Query) Reset()         { *m = Query{} }
func (m *Query) String() string { return proto.CompactTextString(m) }
func (*Query) ProtoMessage()    {}
func (*Query) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{1}
}

func (m *Query) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Query.Unmarshal(m, b)
}
func (m *Query) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Query.Marshal(b, m, deterministic)
}
func (m *Query) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Query.Merge(m, src)
}
func (m *Query) XXX_Size() int {
	return xxx_messageInfo_Query.Size(m)
}
func (m *Query) XXX_DiscardUnknown() {
	xxx_messageInfo_Query.DiscardUnknown(m)
}

var xxx_messageInfo_Query proto.InternalMessageInfo

func (m *Query) GetRefId() string {
	if m != nil {
		return m.RefId
	}
	return ""
}

func (m *Query) GetMaxDataPoints() int64 {
	if m != nil {
		return m.MaxDataPoints
	}
	return 0
}

func (m *Query) GetIntervalMs() int64 {
	if m != nil {
		return m.IntervalMs
	}
	return 0
}

func (m *Query) GetModelJson() string {
	if m != nil {
		return m.ModelJson
	}
	return ""
}

type TimeRange struct {
	FromRaw              string   `protobuf:"bytes,1,opt,name=fromRaw,proto3" json:"fromRaw,omitempty"`
	ToRaw                string   `protobuf:"bytes,2,opt,name=toRaw,proto3" json:"toRaw,omitempty"`
	FromEpochMs          int64    `protobuf:"varint,3,opt,name=fromEpochMs,proto3" json:"fromEpochMs,omitempty"`
	ToEpochMs            int64    `protobuf:"varint,4,opt,name=toEpochMs,proto3" json:"toEpochMs,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TimeRange) Reset()         { *m = TimeRange{} }
func (m *TimeRange) String() string { return proto.CompactTextString(m) }
func (*TimeRange) ProtoMessage()    {}
func (*TimeRange) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{2}
}

func (m *TimeRange) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TimeRange.Unmarshal(m, b)
}
func (m *TimeRange) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TimeRange.Marshal(b, m, deterministic)
}
func (m *TimeRange) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TimeRange.Merge(m, src)
}
func (m *TimeRange) XXX_Size() int {
	return xxx_messageInfo_TimeRange.Size(m)
}
func (m *TimeRange) XXX_DiscardUnknown() {
	xxx_messageInfo_TimeRange.DiscardUnknown(m)
}

var xxx_messageInfo_TimeRange proto.InternalMessageInfo

func (m *TimeRange) GetFromRaw() string {
	if m != nil {
		return m.FromRaw
	}
	return ""
}

func (m *TimeRange) GetToRaw() string {
	if m != nil {
		return m.ToRaw
	}
	return ""
}

func (m *TimeRange) GetFromEpochMs() int64 {
	if m != nil {
		return m.FromEpochMs
	}
	return 0
}

func (m *TimeRange) GetToEpochMs() int64 {
	if m != nil {
		return m.ToEpochMs
	}
	return 0
}

type DatasourceResponse struct {
	Results              []*QueryResult `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *DatasourceResponse) Reset()         { *m = DatasourceResponse{} }
func (m *DatasourceResponse) String() string { return proto.CompactTextString(m) }
func (*DatasourceResponse) ProtoMessage()    {}
func (*DatasourceResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{3}
}

func (m *DatasourceResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DatasourceResponse.Unmarshal(m, b)
}
func (m *DatasourceResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DatasourceResponse.Marshal(b, m, deterministic)
}
func (m *DatasourceResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DatasourceResponse.Merge(m, src)
}
func (m *DatasourceResponse) XXX_Size() int {
	return xxx_messageInfo_DatasourceResponse.Size(m)
}
func (m *DatasourceResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DatasourceResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DatasourceResponse proto.InternalMessageInfo

func (m *DatasourceResponse) GetResults() []*QueryResult {
	if m != nil {
		return m.Results
	}
	return nil
}

type QueryResult struct {
	Error                string        `protobuf:"bytes,1,opt,name=error,proto3" json:"error,omitempty"`
	RefId                string        `protobuf:"bytes,2,opt,name=refId,proto3" json:"refId,omitempty"`
	MetaJson             string        `protobuf:"bytes,3,opt,name=metaJson,proto3" json:"metaJson,omitempty"`
	Series               []*TimeSeries `protobuf:"bytes,4,rep,name=series,proto3" json:"series,omitempty"`
	Tables               []*Table      `protobuf:"bytes,5,rep,name=tables,proto3" json:"tables,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *QueryResult) Reset()         { *m = QueryResult{} }
func (m *QueryResult) String() string { return proto.CompactTextString(m) }
func (*QueryResult) ProtoMessage()    {}
func (*QueryResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{4}
}

func (m *QueryResult) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_QueryResult.Unmarshal(m, b)
}
func (m *QueryResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_QueryResult.Marshal(b, m, deterministic)
}
func (m *QueryResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryResult.Merge(m, src)
}
func (m *QueryResult) XXX_Size() int {
	return xxx_messageInfo_QueryResult.Size(m)
}
func (m *QueryResult) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryResult.DiscardUnknown(m)
}

var xxx_messageInfo_QueryResult proto.InternalMessageInfo

func (m *QueryResult) GetError() string {
	if m != nil {
		return m.Error
	}
	return ""
}

func (m *QueryResult) GetRefId() string {
	if m != nil {
		return m.RefId
	}
	return ""
}

func (m *QueryResult) GetMetaJson() string {
	if m != nil {
		return m.MetaJson
	}
	return ""
}

func (m *QueryResult) GetSeries() []*TimeSeries {
	if m != nil {
		return m.Series
	}
	return nil
}

func (m *QueryResult) GetTables() []*Table {
	if m != nil {
		return m.Tables
	}
	return nil
}

type Table struct {
	Columns              []*TableColumn `protobuf:"bytes,1,rep,name=columns,proto3" json:"columns,omitempty"`
	Rows                 []*TableRow    `protobuf:"bytes,2,rep,name=rows,proto3" json:"rows,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Table) Reset()         { *m = Table{} }
func (m *Table) String() string { return proto.CompactTextString(m) }
func (*Table) ProtoMessage()    {}
func (*Table) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{5}
}

func (m *Table) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Table.Unmarshal(m, b)
}
func (m *Table) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Table.Marshal(b, m, deterministic)
}
func (m *Table) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Table.Merge(m, src)
}
func (m *Table) XXX_Size() int {
	return xxx_messageInfo_Table.Size(m)
}
func (m *Table) XXX_DiscardUnknown() {
	xxx_messageInfo_Table.DiscardUnknown(m)
}

var xxx_messageInfo_Table proto.InternalMessageInfo

func (m *Table) GetColumns() []*TableColumn {
	if m != nil {
		return m.Columns
	}
	return nil
}

func (m *Table) GetRows() []*TableRow {
	if m != nil {
		return m.Rows
	}
	return nil
}

type TableColumn struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TableColumn) Reset()         { *m = TableColumn{} }
func (m *TableColumn) String() string { return proto.CompactTextString(m) }
func (*TableColumn) ProtoMessage()    {}
func (*TableColumn) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{6}
}

func (m *TableColumn) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TableColumn.Unmarshal(m, b)
}
func (m *TableColumn) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TableColumn.Marshal(b, m, deterministic)
}
func (m *TableColumn) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TableColumn.Merge(m, src)
}
func (m *TableColumn) XXX_Size() int {
	return xxx_messageInfo_TableColumn.Size(m)
}
func (m *TableColumn) XXX_DiscardUnknown() {
	xxx_messageInfo_TableColumn.DiscardUnknown(m)
}

var xxx_messageInfo_TableColumn proto.InternalMessageInfo

func (m *TableColumn) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

type TableRow struct {
	Values               []*RowValue `protobuf:"bytes,1,rep,name=values,proto3" json:"values,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *TableRow) Reset()         { *m = TableRow{} }
func (m *TableRow) String() string { return proto.CompactTextString(m) }
func (*TableRow) ProtoMessage()    {}
func (*TableRow) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{7}
}

func (m *TableRow) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TableRow.Unmarshal(m, b)
}
func (m *TableRow) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TableRow.Marshal(b, m, deterministic)
}
func (m *TableRow) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TableRow.Merge(m, src)
}
func (m *TableRow) XXX_Size() int {
	return xxx_messageInfo_TableRow.Size(m)
}
func (m *TableRow) XXX_DiscardUnknown() {
	xxx_messageInfo_TableRow.DiscardUnknown(m)
}

var xxx_messageInfo_TableRow proto.InternalMessageInfo

func (m *TableRow) GetValues() []*RowValue {
	if m != nil {
		return m.Values
	}
	return nil
}

type RowValue struct {
	Type                 RowValue_Type `protobuf:"varint,1,opt,name=type,proto3,enum=models.RowValue_Type" json:"type,omitempty"`
	DoubleValue          float64       `protobuf:"fixed64,2,opt,name=doubleValue,proto3" json:"doubleValue,omitempty"`
	Int64Value           int64         `protobuf:"varint,3,opt,name=int64Value,proto3" json:"int64Value,omitempty"`
	BoolValue            bool          `protobuf:"varint,4,opt,name=boolValue,proto3" json:"boolValue,omitempty"`
	StringValue          string        `protobuf:"bytes,5,opt,name=stringValue,proto3" json:"stringValue,omitempty"`
	BytesValue           []byte        `protobuf:"bytes,6,opt,name=bytesValue,proto3" json:"bytesValue,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *RowValue) Reset()         { *m = RowValue{} }
func (m *RowValue) String() string { return proto.CompactTextString(m) }
func (*RowValue) ProtoMessage()    {}
func (*RowValue) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{8}
}

func (m *RowValue) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RowValue.Unmarshal(m, b)
}
func (m *RowValue) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RowValue.Marshal(b, m, deterministic)
}
func (m *RowValue) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RowValue.Merge(m, src)
}
func (m *RowValue) XXX_Size() int {
	return xxx_messageInfo_RowValue.Size(m)
}
func (m *RowValue) XXX_DiscardUnknown() {
	xxx_messageInfo_RowValue.DiscardUnknown(m)
}

var xxx_messageInfo_RowValue proto.InternalMessageInfo

func (m *RowValue) GetType() RowValue_Type {
	if m != nil {
		return m.Type
	}
	return RowValue_TYPE_NULL
}

func (m *RowValue) GetDoubleValue() float64 {
	if m != nil {
		return m.DoubleValue
	}
	return 0
}

func (m *RowValue) GetInt64Value() int64 {
	if m != nil {
		return m.Int64Value
	}
	return 0
}

func (m *RowValue) GetBoolValue() bool {
	if m != nil {
		return m.BoolValue
	}
	return false
}

func (m *RowValue) GetStringValue() string {
	if m != nil {
		return m.StringValue
	}
	return ""
}

func (m *RowValue) GetBytesValue() []byte {
	if m != nil {
		return m.BytesValue
	}
	return nil
}

type DatasourceInfo struct {
	Id                      int64             `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	OrgId                   int64             `protobuf:"varint,2,opt,name=orgId,proto3" json:"orgId,omitempty"`
	Name                    string            `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Type                    string            `protobuf:"bytes,4,opt,name=type,proto3" json:"type,omitempty"`
	Url                     string            `protobuf:"bytes,5,opt,name=url,proto3" json:"url,omitempty"`
	JsonData                string            `protobuf:"bytes,6,opt,name=jsonData,proto3" json:"jsonData,omitempty"`
	DecryptedSecureJsonData map[string]string `protobuf:"bytes,7,rep,name=decryptedSecureJsonData,proto3" json:"decryptedSecureJsonData,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral    struct{}          `json:"-"`
	XXX_unrecognized        []byte            `json:"-"`
	XXX_sizecache           int32             `json:"-"`
}

func (m *DatasourceInfo) Reset()         { *m = DatasourceInfo{} }
func (m *DatasourceInfo) String() string { return proto.CompactTextString(m) }
func (*DatasourceInfo) ProtoMessage()    {}
func (*DatasourceInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{9}
}

func (m *DatasourceInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DatasourceInfo.Unmarshal(m, b)
}
func (m *DatasourceInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DatasourceInfo.Marshal(b, m, deterministic)
}
func (m *DatasourceInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DatasourceInfo.Merge(m, src)
}
func (m *DatasourceInfo) XXX_Size() int {
	return xxx_messageInfo_DatasourceInfo.Size(m)
}
func (m *DatasourceInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_DatasourceInfo.DiscardUnknown(m)
}

var xxx_messageInfo_DatasourceInfo proto.InternalMessageInfo

func (m *DatasourceInfo) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *DatasourceInfo) GetOrgId() int64 {
	if m != nil {
		return m.OrgId
	}
	return 0
}

func (m *DatasourceInfo) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *DatasourceInfo) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *DatasourceInfo) GetUrl() string {
	if m != nil {
		return m.Url
	}
	return ""
}

func (m *DatasourceInfo) GetJsonData() string {
	if m != nil {
		return m.JsonData
	}
	return ""
}

func (m *DatasourceInfo) GetDecryptedSecureJsonData() map[string]string {
	if m != nil {
		return m.DecryptedSecureJsonData
	}
	return nil
}

type TimeSeries struct {
	Name                 string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Tags                 map[string]string `protobuf:"bytes,2,rep,name=tags,proto3" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Points               []*Point          `protobuf:"bytes,3,rep,name=points,proto3" json:"points,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *TimeSeries) Reset()         { *m = TimeSeries{} }
func (m *TimeSeries) String() string { return proto.CompactTextString(m) }
func (*TimeSeries) ProtoMessage()    {}
func (*TimeSeries) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{10}
}

func (m *TimeSeries) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TimeSeries.Unmarshal(m, b)
}
func (m *TimeSeries) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TimeSeries.Marshal(b, m, deterministic)
}
func (m *TimeSeries) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TimeSeries.Merge(m, src)
}
func (m *TimeSeries) XXX_Size() int {
	return xxx_messageInfo_TimeSeries.Size(m)
}
func (m *TimeSeries) XXX_DiscardUnknown() {
	xxx_messageInfo_TimeSeries.DiscardUnknown(m)
}

var xxx_messageInfo_TimeSeries proto.InternalMessageInfo

func (m *TimeSeries) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *TimeSeries) GetTags() map[string]string {
	if m != nil {
		return m.Tags
	}
	return nil
}

func (m *TimeSeries) GetPoints() []*Point {
	if m != nil {
		return m.Points
	}
	return nil
}

type Point struct {
	Timestamp            int64    `protobuf:"varint,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Value                float64  `protobuf:"fixed64,2,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Point) Reset()         { *m = Point{} }
func (m *Point) String() string { return proto.CompactTextString(m) }
func (*Point) ProtoMessage()    {}
func (*Point) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{11}
}

func (m *Point) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Point.Unmarshal(m, b)
}
func (m *Point) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Point.Marshal(b, m, deterministic)
}
func (m *Point) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Point.Merge(m, src)
}
func (m *Point) XXX_Size() int {
	return xxx_messageInfo_Point.Size(m)
}
func (m *Point) XXX_DiscardUnknown() {
	xxx_messageInfo_Point.DiscardUnknown(m)
}

var xxx_messageInfo_Point proto.InternalMessageInfo

func (m *Point) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Point) GetValue() float64 {
	if m != nil {
		return m.Value
	}
	return 0
}

type QueryDatasourceRequest struct {
	TimeRange            *TimeRange `protobuf:"bytes,1,opt,name=timeRange,proto3" json:"timeRange,omitempty"`
	DatasourceId         int64      `protobuf:"varint,2,opt,name=datasourceId,proto3" json:"datasourceId,omitempty"`
	Queries              []*Query   `protobuf:"bytes,3,rep,name=queries,proto3" json:"queries,omitempty"`
	OrgId                int64      `protobuf:"varint,4,opt,name=orgId,proto3" json:"orgId,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *QueryDatasourceRequest) Reset()         { *m = QueryDatasourceRequest{} }
func (m *QueryDatasourceRequest) String() string { return proto.CompactTextString(m) }
func (*QueryDatasourceRequest) ProtoMessage()    {}
func (*QueryDatasourceRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{12}
}

func (m *QueryDatasourceRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_QueryDatasourceRequest.Unmarshal(m, b)
}
func (m *QueryDatasourceRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_QueryDatasourceRequest.Marshal(b, m, deterministic)
}
func (m *QueryDatasourceRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryDatasourceRequest.Merge(m, src)
}
func (m *QueryDatasourceRequest) XXX_Size() int {
	return xxx_messageInfo_QueryDatasourceRequest.Size(m)
}
func (m *QueryDatasourceRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryDatasourceRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryDatasourceRequest proto.InternalMessageInfo

func (m *QueryDatasourceRequest) GetTimeRange() *TimeRange {
	if m != nil {
		return m.TimeRange
	}
	return nil
}

func (m *QueryDatasourceRequest) GetDatasourceId() int64 {
	if m != nil {
		return m.DatasourceId
	}
	return 0
}

func (m *QueryDatasourceRequest) GetQueries() []*Query {
	if m != nil {
		return m.Queries
	}
	return nil
}

func (m *QueryDatasourceRequest) GetOrgId() int64 {
	if m != nil {
		return m.OrgId
	}
	return 0
}

type QueryDatasourceResponse struct {
	Results              []*QueryResult `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *QueryDatasourceResponse) Reset()         { *m = QueryDatasourceResponse{} }
func (m *QueryDatasourceResponse) String() string { return proto.CompactTextString(m) }
func (*QueryDatasourceResponse) ProtoMessage()    {}
func (*QueryDatasourceResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb096a9d85d590d2, []int{13}
}

func (m *QueryDatasourceResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_QueryDatasourceResponse.Unmarshal(m, b)
}
func (m *QueryDatasourceResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_QueryDatasourceResponse.Marshal(b, m, deterministic)
}
func (m *QueryDatasourceResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryDatasourceResponse.Merge(m, src)
}
func (m *QueryDatasourceResponse) XXX_Size() int {
	return xxx_messageInfo_QueryDatasourceResponse.Size(m)
}
func (m *QueryDatasourceResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryDatasourceResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryDatasourceResponse proto.InternalMessageInfo

func (m *QueryDatasourceResponse) GetResults() []*QueryResult {
	if m != nil {
		return m.Results
	}
	return nil
}

func init() {
	proto.RegisterEnum("models.RowValue_Type", RowValue_Type_name, RowValue_Type_value)
	proto.RegisterType((*DatasourceRequest)(nil), "models.DatasourceRequest")
	proto.RegisterType((*Query)(nil), "models.Query")
	proto.RegisterType((*TimeRange)(nil), "models.TimeRange")
	proto.RegisterType((*DatasourceResponse)(nil), "models.DatasourceResponse")
	proto.RegisterType((*QueryResult)(nil), "models.QueryResult")
	proto.RegisterType((*Table)(nil), "models.Table")
	proto.RegisterType((*TableColumn)(nil), "models.TableColumn")
	proto.RegisterType((*TableRow)(nil), "models.TableRow")
	proto.RegisterType((*RowValue)(nil), "models.RowValue")
	proto.RegisterType((*DatasourceInfo)(nil), "models.DatasourceInfo")
	proto.RegisterMapType((map[string]string)(nil), "models.DatasourceInfo.DecryptedSecureJsonDataEntry")
	proto.RegisterType((*TimeSeries)(nil), "models.TimeSeries")
	proto.RegisterMapType((map[string]string)(nil), "models.TimeSeries.TagsEntry")
	proto.RegisterType((*Point)(nil), "models.Point")
	proto.RegisterType((*QueryDatasourceRequest)(nil), "models.QueryDatasourceRequest")
	proto.RegisterType((*QueryDatasourceResponse)(nil), "models.QueryDatasourceResponse")
}

func init() { proto.RegisterFile("datasource.proto", fileDescriptor_bb096a9d85d590d2) }

var fileDescriptor_bb096a9d85d590d2 = []byte{
	// 922 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x56, 0x5d, 0x6f, 0xe3, 0x44,
	0x14, 0xc5, 0x1f, 0x49, 0xeb, 0x9b, 0xb6, 0xeb, 0x1d, 0x60, 0x37, 0x44, 0xd5, 0x12, 0xac, 0x45,
	0x04, 0x24, 0x02, 0xca, 0x56, 0x05, 0x81, 0x84, 0x44, 0xb7, 0xd1, 0x92, 0xaa, 0xb4, 0x65, 0x9a,
	0x45, 0x5a, 0x84, 0x04, 0x4e, 0x32, 0x09, 0x06, 0xdb, 0x93, 0x9d, 0x19, 0xb7, 0x44, 0x3c, 0xf1,
	0x6f, 0x78, 0xe0, 0x89, 0x3f, 0xc0, 0x03, 0x0f, 0xfc, 0x2d, 0x34, 0x77, 0xfc, 0x95, 0xa6, 0x8b,
	0x00, 0xf1, 0xe6, 0x7b, 0xce, 0xbd, 0x77, 0x8e, 0xef, 0x9c, 0x19, 0x1b, 0xfc, 0x59, 0xa8, 0x42,
	0xc9, 0x33, 0x31, 0x65, 0xfd, 0xa5, 0xe0, 0x8a, 0x93, 0x66, 0xc2, 0x67, 0x2c, 0x96, 0xc1, 0x1f,
	0x16, 0xdc, 0x3d, 0x2e, 0x49, 0xca, 0x9e, 0x67, 0x4c, 0x2a, 0xf2, 0x1e, 0x78, 0x2a, 0x4a, 0x18,
	0x0d, 0xd3, 0x05, 0x6b, 0x5b, 0x5d, 0xab, 0xd7, 0x1a, 0xdc, 0xed, 0x9b, 0x8a, 0xfe, 0xb8, 0x20,
	0x68, 0x95, 0x43, 0x0e, 0x01, 0xaa, 0x25, 0xda, 0x36, 0x56, 0xdc, 0x2b, 0x2a, 0xaa, 0xfe, 0xa3,
	0x74, 0xce, 0x69, 0x2d, 0x93, 0xbc, 0x05, 0x5b, 0xcf, 0x33, 0x26, 0x22, 0x26, 0xdb, 0x4e, 0xd7,
	0xe9, 0xb5, 0x06, 0xbb, 0x45, 0xd1, 0x17, 0x19, 0x13, 0x2b, 0x5a, 0xb0, 0x64, 0x1f, 0x3c, 0x61,
	0xc4, 0x8d, 0x66, 0x6d, 0xb7, 0x6b, 0xf5, 0x76, 0x69, 0x05, 0x04, 0x3f, 0x5b, 0xd0, 0xc0, 0x02,
	0xf2, 0x0a, 0x34, 0x04, 0x9b, 0x8f, 0x66, 0xa8, 0xda, 0xa3, 0x26, 0x20, 0x0f, 0x61, 0x37, 0x09,
	0x7f, 0xd4, 0x3a, 0x2e, 0x78, 0x94, 0x2a, 0x89, 0x0a, 0x1d, 0xba, 0x0e, 0x92, 0x07, 0x00, 0x51,
	0xaa, 0x98, 0xb8, 0x0a, 0xe3, 0xcf, 0xb5, 0x1e, 0x9d, 0x52, 0x43, 0xb4, 0x06, 0x14, 0x77, 0x22,
	0x79, 0x8a, 0x1a, 0x3c, 0x5a, 0x01, 0xc1, 0x4f, 0xe0, 0x95, 0xa3, 0x21, 0x6d, 0xd8, 0x9a, 0x0b,
	0x9e, 0xd0, 0xf0, 0x3a, 0x17, 0x52, 0x84, 0x5a, 0xa0, 0xe2, 0x1a, 0xb7, 0x8d, 0x40, 0x0c, 0x48,
	0x17, 0x5a, 0x3a, 0x61, 0xb8, 0xe4, 0xd3, 0xef, 0xca, 0xb5, 0xeb, 0x90, 0x5e, 0x5c, 0xf1, 0x82,
	0x77, 0x91, 0xaf, 0x80, 0xe0, 0x31, 0x90, 0xfa, 0x2e, 0xca, 0x25, 0x4f, 0x25, 0x23, 0xef, 0xc2,
	0x96, 0x60, 0x32, 0x8b, 0x95, 0x6c, 0x5b, 0x38, 0xdd, 0x97, 0xd7, 0xa7, 0x8b, 0x1c, 0x2d, 0x72,
	0x82, 0x5f, 0x2c, 0x68, 0xd5, 0x08, 0x2d, 0x95, 0x09, 0xc1, 0x45, 0x31, 0x4b, 0x0c, 0xaa, 0x09,
	0xdb, 0xf5, 0x09, 0x77, 0x60, 0x3b, 0x61, 0x2a, 0xc4, 0xd1, 0x38, 0x48, 0x94, 0x31, 0x79, 0x07,
	0x9a, 0xd2, 0xec, 0xb1, 0x8b, 0x2a, 0x48, 0xdd, 0x4a, 0x97, 0xc8, 0xd0, 0x3c, 0x83, 0xbc, 0x09,
	0x4d, 0x15, 0x4e, 0x62, 0x26, 0xdb, 0x8d, 0x75, 0x3f, 0x8c, 0x35, 0x4a, 0x73, 0x32, 0xf8, 0x1a,
	0x1a, 0x08, 0xe8, 0x57, 0x9c, 0xf2, 0x38, 0x4b, 0xd2, 0x8d, 0x57, 0x44, 0xfe, 0x31, 0x72, 0xb4,
	0xc8, 0x21, 0x0f, 0xc1, 0x15, 0xfc, 0x5a, 0xef, 0xbf, 0xce, 0xf5, 0xd7, 0x9b, 0xf3, 0x6b, 0x8a,
	0x6c, 0xf0, 0x06, 0xb4, 0x6a, 0xd5, 0x84, 0x80, 0x9b, 0x86, 0x09, 0xcb, 0xc7, 0x80, 0xcf, 0xc1,
	0x01, 0x6c, 0x17, 0x45, 0xa4, 0x07, 0xcd, 0xab, 0x30, 0xce, 0x58, 0x21, 0xa1, 0x6c, 0x4b, 0xf9,
	0xf5, 0x97, 0x9a, 0xa0, 0x39, 0x1f, 0xfc, 0x6e, 0xc3, 0x76, 0x01, 0x92, 0xb7, 0xc1, 0x55, 0xab,
	0xa5, 0x69, 0xbb, 0x37, 0x78, 0xf5, 0x66, 0x51, 0x7f, 0xbc, 0x5a, 0x32, 0x8a, 0x29, 0xda, 0x1e,
	0x33, 0x9e, 0x4d, 0x62, 0x86, 0x0c, 0x4e, 0xde, 0xa2, 0x75, 0x28, 0xf7, 0xee, 0xe1, 0x81, 0x49,
	0xa8, 0xbc, 0x9b, 0x23, 0xda, 0x3e, 0x13, 0xce, 0x63, 0x43, 0x6b, 0xfb, 0x6c, 0xd3, 0x0a, 0xd0,
	0xfd, 0xa5, 0x12, 0x51, 0xba, 0x30, 0x7c, 0x03, 0x5f, 0xb4, 0x0e, 0xe9, 0xfe, 0x93, 0x95, 0x62,
	0xd2, 0x24, 0x34, 0xbb, 0x56, 0x6f, 0x87, 0xd6, 0x90, 0x60, 0x0e, 0xae, 0xd6, 0x4b, 0x76, 0xc1,
	0x1b, 0x3f, 0xbb, 0x18, 0x7e, 0x73, 0xf6, 0xf4, 0xf4, 0xd4, 0x7f, 0x89, 0xdc, 0x81, 0x16, 0x86,
	0xc7, 0xe7, 0x4f, 0x8f, 0x4e, 0x87, 0xbe, 0x45, 0xf6, 0x00, 0x10, 0x18, 0x9d, 0x8d, 0x0f, 0x0f,
	0x7c, 0xbb, 0xcc, 0x3f, 0x3a, 0x3f, 0x3f, 0xf5, 0x9d, 0x32, 0xff, 0x72, 0x4c, 0x47, 0x67, 0x4f,
	0x7c, 0xb7, 0xcc, 0x3f, 0x7a, 0x36, 0x1e, 0x5e, 0xfa, 0x8d, 0xe0, 0x4f, 0x1b, 0xf6, 0xd6, 0xef,
	0x13, 0xb2, 0x07, 0x76, 0x64, 0xce, 0xbb, 0x43, 0xed, 0x68, 0xa6, 0x0d, 0xca, 0xc5, 0x22, 0x37,
	0xa8, 0x43, 0x4d, 0x50, 0x6e, 0xa2, 0x53, 0x6d, 0xa2, 0xc6, 0x70, 0x07, 0xcc, 0x59, 0x36, 0xa3,
	0xf6, 0xc1, 0xc9, 0x44, 0x9c, 0x8f, 0x40, 0x3f, 0x6a, 0x6b, 0x7f, 0x2f, 0x79, 0xaa, 0x57, 0xc5,
	0x17, 0xf7, 0x68, 0x19, 0x93, 0x04, 0xee, 0xcf, 0xd8, 0x54, 0xac, 0x96, 0x8a, 0xcd, 0x2e, 0xd9,
	0x34, 0x13, 0xec, 0xa4, 0x48, 0xdd, 0x42, 0x2f, 0x3c, 0xba, 0xfd, 0x12, 0xec, 0x1f, 0xdf, 0x5e,
	0x35, 0x4c, 0x95, 0x58, 0xd1, 0x17, 0xf5, 0xec, 0x9c, 0xc0, 0xfe, 0xdf, 0x15, 0x6a, 0xf1, 0x3f,
	0xb0, 0x55, 0x6e, 0x54, 0xfd, 0xa8, 0x87, 0x71, 0x55, 0x7a, 0xc6, 0xa3, 0x26, 0xf8, 0xc8, 0xfe,
	0xd0, 0x0a, 0x7e, 0xb3, 0x00, 0xaa, 0x03, 0x78, 0x9b, 0xc9, 0xc9, 0xfb, 0xe0, 0xaa, 0x70, 0x51,
	0x9c, 0x96, 0xfd, 0xcd, 0x63, 0xdb, 0x1f, 0x87, 0x0b, 0x69, 0x34, 0x63, 0xa6, 0x3e, 0xbe, 0x4b,
	0x73, 0xc3, 0xde, 0xb8, 0xce, 0xf1, 0x8a, 0xa5, 0x39, 0xd9, 0xf9, 0x00, 0xbc, 0xb2, 0xf2, 0x5f,
	0x89, 0xfe, 0x18, 0x1a, 0xd8, 0x09, 0xaf, 0xc3, 0x28, 0x61, 0x52, 0x85, 0xc9, 0x32, 0xdf, 0xfb,
	0x0a, 0x58, 0x6f, 0x60, 0xe5, 0x0d, 0x82, 0x5f, 0x2d, 0xb8, 0x87, 0xf7, 0xdb, 0xff, 0xf0, 0xc1,
	0x0b, 0x60, 0xa7, 0xfa, 0x8c, 0x95, 0x5e, 0x5b, 0xc3, 0xfe, 0xf9, 0xc7, 0xad, 0x74, 0xac, 0x5b,
	0x73, 0x6c, 0xf0, 0x19, 0xdc, 0xdf, 0x50, 0xfb, 0x9f, 0x2e, 0xf6, 0x01, 0x05, 0xbf, 0x6a, 0x72,
	0x11, 0x67, 0x8b, 0x28, 0x25, 0x9f, 0x14, 0x5f, 0xcc, 0xd7, 0x36, 0x1d, 0x9a, 0x4f, 0xa5, 0xd3,
	0xb9, 0x8d, 0x32, 0x12, 0x06, 0xdf, 0x02, 0x3c, 0x11, 0xe1, 0x3c, 0x4c, 0xc3, 0x4f, 0x2f, 0x46,
	0x84, 0xc2, 0x9d, 0x1b, 0x5a, 0xc9, 0x83, 0x35, 0x49, 0x9b, 0xcd, 0x5f, 0x7f, 0x21, 0x6f, 0x56,
	0x38, 0xda, 0xf9, 0xaa, 0xf6, 0xa7, 0x30, 0x69, 0xe2, 0x7f, 0xcb, 0xa3, 0xbf, 0x02, 0x00, 0x00,
	0xff, 0xff, 0x3d, 0x9a, 0x09, 0x2c, 0xcb, 0x08, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// DatasourcePluginClient is the client API for DatasourcePlugin service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type DatasourcePluginClient interface {
	Query(ctx context.Context, in *DatasourceRequest, opts ...grpc.CallOption) (*DatasourceResponse, error)
}

type datasourcePluginClient struct {
	cc *grpc.ClientConn
}

func NewDatasourcePluginClient(cc *grpc.ClientConn) DatasourcePluginClient {
	return &datasourcePluginClient{cc}
}

func (c *datasourcePluginClient) Query(ctx context.Context, in *DatasourceRequest, opts ...grpc.CallOption) (*DatasourceResponse, error) {
	out := new(DatasourceResponse)
	err := c.cc.Invoke(ctx, "/models.DatasourcePlugin/Query", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DatasourcePluginServer is the server API for DatasourcePlugin service.
type DatasourcePluginServer interface {
	Query(context.Context, *DatasourceRequest) (*DatasourceResponse, error)
}

// UnimplementedDatasourcePluginServer can be embedded to have forward compatible implementations.
type UnimplementedDatasourcePluginServer struct {
}

func (*UnimplementedDatasourcePluginServer) Query(ctx context.Context, req *DatasourceRequest) (*DatasourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Query not implemented")
}

func RegisterDatasourcePluginServer(s *grpc.Server, srv DatasourcePluginServer) {
	s.RegisterService(&_DatasourcePlugin_serviceDesc, srv)
}

func _DatasourcePlugin_Query_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DatasourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatasourcePluginServer).Query(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/models.DatasourcePlugin/Query",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatasourcePluginServer).Query(ctx, req.(*DatasourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _DatasourcePlugin_serviceDesc = grpc.ServiceDesc{
	ServiceName: "models.DatasourcePlugin",
	HandlerType: (*DatasourcePluginServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Query",
			Handler:    _DatasourcePlugin_Query_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "datasource.proto",
}

// GrafanaAPIClient is the client API for GrafanaAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type GrafanaAPIClient interface {
	QueryDatasource(ctx context.Context, in *QueryDatasourceRequest, opts ...grpc.CallOption) (*QueryDatasourceResponse, error)
}

type grafanaAPIClient struct {
	cc *grpc.ClientConn
}

func NewGrafanaAPIClient(cc *grpc.ClientConn) GrafanaAPIClient {
	return &grafanaAPIClient{cc}
}

func (c *grafanaAPIClient) QueryDatasource(ctx context.Context, in *QueryDatasourceRequest, opts ...grpc.CallOption) (*QueryDatasourceResponse, error) {
	out := new(QueryDatasourceResponse)
	err := c.cc.Invoke(ctx, "/models.GrafanaAPI/QueryDatasource", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GrafanaAPIServer is the server API for GrafanaAPI service.
type GrafanaAPIServer interface {
	QueryDatasource(context.Context, *QueryDatasourceRequest) (*QueryDatasourceResponse, error)
}

// UnimplementedGrafanaAPIServer can be embedded to have forward compatible implementations.
type UnimplementedGrafanaAPIServer struct {
}

func (*UnimplementedGrafanaAPIServer) QueryDatasource(ctx context.Context, req *QueryDatasourceRequest) (*QueryDatasourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryDatasource not implemented")
}

func RegisterGrafanaAPIServer(s *grpc.Server, srv GrafanaAPIServer) {
	s.RegisterService(&_GrafanaAPI_serviceDesc, srv)
}

func _GrafanaAPI_QueryDatasource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDatasourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GrafanaAPIServer).QueryDatasource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/models.GrafanaAPI/QueryDatasource",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GrafanaAPIServer).QueryDatasource(ctx, req.(*QueryDatasourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _GrafanaAPI_serviceDesc = grpc.ServiceDesc{
	ServiceName: "models.GrafanaAPI",
	HandlerType: (*GrafanaAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "QueryDatasource",
			Handler:    _GrafanaAPI_QueryDatasource_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "datasource.proto",
}
