# Logs

Status: EARLY Draft/Proposal

## LogLines

### Properties and field requirements

- **Time field** - _required_
  - There must be at least one non nullable time field
  - If there are multiple time fields present, following will decide the priority
    - field of type `Nano second time field` described later in the document
    - or first matching time field with name `timestamp`
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
  - Log level can be one of the values specified in the docs [here](https://grafana.com/docs/grafana/latest/packages_api/data/loglevel/#enumeration-members)
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
  - Field type must be nullable string which represent a JSON object. Example value: `{}`, `{"hello":"world", "foo": 123.45, "bar" :["yellow","red"], "baz" : { "name": "alice" }}`
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

- **Any other fields related to context such as trace id, span id etc**

## Meta data requirements

- Contract doesn't require any specific meta data.
- Frame meta can optionally specify `preferredVisualisationType:logs` as meta data. Without this property, explore page will be rendering the logs data as table instead in logs view

## Example

TBD

## Invalid cases

- Frame without time field
- Frame without string field
- Frame with field name "tsNs" where the type of the "tsNs" field is not string.

## Useful links

- [OTel Logs Data Model](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/data-model.md)
- [OTel Logs Level](https://docs.google.com/document/d/1WQDz1jF0yKBXe3OibXWfy3g6lor9SvjZ4xT-8uuDCiA/edit#)
- [Javascript high resolution timestamp](https://www.w3.org/TR/hr-time/)
