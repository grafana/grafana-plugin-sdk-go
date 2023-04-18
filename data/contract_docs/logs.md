<!-- markdownlint-configure-file {
  "MD013": false,
  "MD033": false
} -->

# Logs

Status: EARLY Draft/Proposal

## LogLines

Version: 0.0

### Properties and field requirements

- **Time field** - _required_
  - There must be at least one non nullable time field
  - If there are multiple time fields present, following will decide the priority
    - First matching time field with name `timestamp`
      - If there are multiple time fields, and none of them is named `timestamp`, it is considered an error.
- **Message field** - _required_
  - There must be at lease one non nullable string field must present
  - If more than one string fields found, the following will decide the priority
    - First matching string field with name `body`
      - If there are multiple string fields, and none of them is named `body`, it is considered an error.
- **Severity field** - _optional_
  - This is optional field
  - Level/Severity of the log line can be represented with this field.
  - First matching string field with name `severity` will be considered as severity field
  - If no level field found, consumers/client will decide the log level. Example: logs panel will try to parse the message field and determine the log level
  - Log level can be one of the values specified in the docs [here](https://grafana.com/docs/grafana/latest/explore/logs-integration/)
- **ID field** - _optional_
  - This optional field
  - Unique identified of the log line
  - This have to be a string field. (either nullable or non-nullable string field)
  - First matching string field with name `id` will be considered as severity field
  - If no id field found, consumers/client will decide the id field as required.
- **Attributes field** - _optional_
  - This is an optional field
  - This field is also known as labels
  - This field represent additional attributes of the log line. This is also known as labels field.
  - Field type must be json raw message type. Example value: `{}`, `{"hello":"world", "foo": 123.45, "bar" :["yellow","red"], "baz" : { "name": "alice" }}`
    - Should not be empty string.
    - Value should be represented with `Record<string,any>` type in javascript.
  - First matching string field with name `attributes` will be considered as attributes field

Any other field is ignored.

## Example

Following is an example of a logs frame in go

```go
data.NewFrame(
    "logs",
    data.NewField("timestamp", nil, []time.Time{time.UnixMilli(1645030244810), time.UnixMilli(1645030247027), time.UnixMilli(1645030247027)}),
    data.NewField("body", nil, []string{"message one", "message two", "message three"}),
    data.NewField("severity", nil, []string{"critical", "error", "warning"}),
    data.NewField("id", nil, []string{"xxx-001", "xyz-002", "111-003"}),
    data.NewField("attributes", nil, []json.RawMessage{[]byte(`{}`), []byte(`{"hello":"world"}`), []byte(`{"hello":"world", "foo": 123.45, "bar" :["yellow","red"], "baz" : { "name": "alice" }}`)}),
)
```

the same can be represented as

| Name: timestamp <br/> Type: []time.Time | Name: body <br/> Type: []string | Name: severity <br/> Type: []\*string | Name: id <br/> Type: []\*string | Name: attributes <br/> Type: []json.RawMessage                                         |
| --------------------------------------- | ------------------------------- | ------------------------------------- | ------------------------------- | -------------------------------------------------------------------------------------- |
| 2022-02-16 16:50:44.810 +0000 GMT       | message one                     | critical                              | xxx-001                         | {}                                                                                     |
| 2022-02-16 16:50:47.027 +0000 GMT       | message two                     | error                                 | xyz-002                         | {"hello":"world"}                                                                      |
| 2022-02-16 16:50:47.027 +0000 GMT       | message three                   | warning                               | 111-003                         | {"hello":"world", "foo": 123.45, "bar" :["yellow","red"], "baz" : { "name": "alice" }} |

## Meta data requirements

- Frame type must be set to `FrameTypeLogLines`/`log-lines`
- Frame meta can optionally specify `preferredVisualisationType:logs` as meta data. Without this property, explore page will be rendering the logs data as table instead in logs view

## Invalid cases

- Frame without time field
- Frame without string field
- Frame with field name "tsNs" where the type of the "tsNs" field is not number.

## Useful links

- [OTel Logs Data Model](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/data-model.md)
- [OTel Logs Level](https://docs.google.com/document/d/1WQDz1jF0yKBXe3OibXWfy3g6lor9SvjZ4xT-8uuDCiA/edit#)
- [Javascript high resolution timestamp](https://www.w3.org/TR/hr-time/)
