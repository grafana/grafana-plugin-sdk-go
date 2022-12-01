# Logs

Status: EARLY Draft/Proposal

## LogLines

### Properties and field requirements

- **Time field** - _required_
  - There must be at least one non nullable time field
  - If there are multiple time fields present, following will decide the priority
    - First matching time field with name `timestamp`
    - or first matching time field with name `ts`
    - or first matching time field
- **Message field** - _required_
  - There must be at lease one non nullable string field must present
  - If more than one string fields found, the following will decide the priority
    - First matching string field with name `body`
    - or first matching string field with name `message`
    - or first matching string field
- **Severity field** - _optional_
  - This is optional field
  - Level/Severity of the log line can be represented with this field.
  - This have to be a string field. (either nullable or non-nullable string field)
  - Log field will be decided in the following order
    - First matching string field with name `severity`
    - or first matching string field with name `level`
  - If no level field found, consumers/client will decide the log level. Example: logs panels will try to parse the message field and determine the log level
  - Log level can be one of the values specified in the docs [here](https://grafana.com/docs/grafana/latest/explore/logs-integration/)
- **ID field** - _optional_
  - This optional field
  - Unique identified of the log line
  - This have to be a string field. (either nullable or non-nullable string field)
  - Id field will be decided in the following order
    - First matching string field with name `id`
    - or first matching string field with name `guid`
  - If no id field found, consumers/client will decide the id field as required.
- **Attributes field** - _optional_ / **Labels field**
  - This is an optional field
  - This field represent additional attributes of the log line. This is also known as labels field.
  - Field type must be json raw message type. Example value: `{}`, `{"hello":"world", "foo": 123.45, "bar" :["yellow","red"], "baz" : { "name": "alice" }}`
    - Should not be empty string.
    - Value should be represented with `Record<string|number,any>` type in javascript.
  - Attribute field will be decided in the following order
    - First matching string field with name `attributes`
    - or first matching string field with name `labels`

### Additional fields

- **NanoSecond Time field** - _optional_

  - When the log line have sub-milli second precisions, regular time field not suitable to represent them.
  - Field type must be non-nullable string and field name must be `tsNs`
  - when this field detected, clients are suggested to use this field and ignore the main time field
  - Field values can't be empty and must have only positive numbers. (no decimal places or floating numbers)

If any other fields (remainder fields) found, they will be treated as items of the attributes field.

## Example

Following is an example of a logs frame in go

```go
data.NewFrame("logs",
    data.NewField("timestamp", nil, []time.Time{time.UnixMilli(1645030244810), time.UnixMilli(1645030247027), time.UnixMilli(1645030247027)}),
    data.NewField("body", nil, []string{"message one", "message two", "message three"}),
    data.NewField("severity", nil, []string{"critical", "error", "warning"}),
    data.NewField("id", nil, []string{"xxx-001", "xyz-002", "111-003"}),
    data.NewField("attributes", nil, []json.RawMessage{[]byte(`{}`), []byte(`{"hello":"world"}`), []byte(`{"hello":"world", "foo": 123.45, "bar" :["yellow","red"], "baz" : { "name": "alice" }}`)}),
    data.NewField("tsNs", nil, []string{"1645030244810757120", "1645030247027735040", "1645030247027745040"}),
)
```

the same can be represented as

| Name: timestamp <br/> Type: []time.Time | Name: body <br/> Type: []string | Name: severity <br/> Type: []string | Name: id <br/> Type: []string | Name: attributes <br/> Type: []json.RawMessage                                         | Name: tsNs <br/> Type: []string |
| --------------------------------------- | ------------------------------- | ----------------------------------- | ----------------------------- | -------------------------------------------------------------------------------------- | ------------------------------- |
| 2022-02-16 16:50:44.810 +0000 GMT       | message one                     | critical                            | xxx-001                       | {}                                                                                     | 1645030244810757120             |
| 2022-02-16 16:50:47.027 +0000 GMT       | message two                     | error                               | xyz-002                       | {"hello":"world"}                                                                      | 1645030247027735040             |
| 2022-02-16 16:50:47.027 +0000 GMT       | message three                   | warning                             | 111-003                       | {"hello":"world", "foo": 123.45, "bar" :["yellow","red"], "baz" : { "name": "alice" }} | 1645030247027745040             |

## Meta data requirements

- Contract doesn't require any specific meta data.
- Frame meta can optionally specify `preferredVisualisationType:logs` as meta data. Without this property, explore page will be rendering the logs data as table instead in logs view

## Invalid cases

- Frame without time field
- Frame without string field
- Frame with field name "tsNs" where the type of the "tsNs" field is not string.

## Useful links

- [OTel Logs Data Model](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/data-model.md)
- [OTel Logs Level](https://docs.google.com/document/d/1WQDz1jF0yKBXe3OibXWfy3g6lor9SvjZ4xT-8uuDCiA/edit#)
- [Javascript high resolution timestamp](https://www.w3.org/TR/hr-time/)
