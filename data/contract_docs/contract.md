# Data Plane Contract - Technical Specification

Status: Draft
Version: 0.1

## Doc Objective

Define in detail common query response schemas for data returned from data sources. This improves the experience for developers of both features and datasources. This will also improve the experience for users through more reliability and quality - which leads to more development time spent more towards improving experience.

Current Backend [proof of concept code](https://github.com/grafana/grafana-plugin-sdk-go/pull/440).

## Kinds and Formats

There are logical **_kinds_** (like Time Series Data, Numeric, Histogram, etc), and there are **_formats_** that a kind can be in.

A **_data type_** definition or declaration in this framework includes both a kind and format. For example, "TimeSeriesWide" is: kind: "Time Series", format: "Wide".

* [Time series](./timeseries.md)
  * [Wide](./timeseries.md#time-series-wide-format-timeserieswide)
  * [Long](./timeseries.md#time-series-long-format-timeserieslong-sql-like)
  * [Multi](./timeseries.md#time-series-multi-format-timeseriesmulti)
* [Numeric](./numeric.md)
  * [Wide](./numeric.md#numeric-wide-format-numericwide)
  * [Multi](./numeric.md#numeric-multi-format-numericmulti)
  * [Long](./numeric.md#numeric-many-format-numericlong)
* [Heatmap](./heatmap.md)
  * [Buckets](./heatmap.md#heatmap-buckets-heatmapbuckets)
  * [Scanlines](./heatmap.md#heatmap-scanlines-heatmapscanlines)
  * [Sparse](./heatmap.md#heatmap-sparse-heatmapsparse)
* [Logs](./logs.md)
  * [LogLines](./logs.md#loglines)

## Dimensional Set Based Kinds

Within a data type (kind+format), there can be multiple **_items_** of data that are uniquely identified. This forms a **_set_** of data items. For example, in the numeric kind there can be a set of numbers, or, in the time series kind, a set of time series-es :-).

Each item of data in a set is uniquely identified by its **_name_** and **_dimensions_**.

Dimensions are facets of data (such as "location" or "host") with a corresponding value. For example, {"host"="a", "location"="new york"}.

Within a dataframe, dimensions are in either a field's Labels property or in string field(s).

### Properties Shared by all Dimensional Set Based Kinds

* When there are multiple items that have the same name, they should have different dimensions (e.g. labels) that uniquely identifies each item[^1].
* The item name should appear in the Name property of each value (numeric or bool typed) Field, as should any Labels[^2]
* A response can have different item names in the response (Note: SSE doesn't currently handle this)

## Remainder Data

Data is encoded into dataframe(s), therefore all types are implemented as an array of data.Frames.

There can be data in dataframe(s) that is not part of the data type's data. This extra data is **_remainder data_**. What readers choose to do with this data is open. However, libraries based on this contract must clearly delineate remainder data from data that is part of the type.

What data becomes remainder data is dependent on and specified in the data type. Generally, it can be additional frames and/or additional fields of a certain field type.

## Invalid Data

Although there is remainder data, there are still cases where the reader should error. The situation for this is when the data type specifier exists on the frame(s), but rules about that type are not followed.

## "No Data" and Empty

There are two named cases for when a response is lacking data and also doesn't have an error.

**_"No Data"_** is for when we retrieve a response from a datasource but the response has no data items. The encoding for the form of a type is a single frame, with the data type declaration, and a zero length of fields (null or []). This is for the case when the entire set has no items. No Data may also be a response with no frames (and no error). However, in this case the Type and other metadata can not be returned -- so the single frame form is usually preferable.

We retrieve one or more data items from a datasource but an item has no values, that item is said to be an "**_Empty value_**". In this case, the required dataframe fields should still be present (but the fields themselves each have no values).

## Error Responses

An error is returned from outside the dataframes using the `Error` and `Status` properties on a [DataResponse](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/backend#DataResponse).

When an error is returned with the DataResponse, a single frame with no fields may be included as well. If the error is present, this will not be considered "No Data". This frame is included so that metadata, in particular a Frame's `ExecutedQueryString`, can be returned to the caller with an error.

Note: In a backend plugin an error can be returned from a [`DataQueryHandler`](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/backend#QueryDataHandler) call. This should only be used when the entire request (all queries) will fail.

## Multi Data Type Responses

The case where a response has multiple data types in a single result (Within a RefID) exists but is currently out of scope for this version of the spec.

However, it needs to be possible to add support for this case. For now, the following logic is suggested:

* Per data type, within a response, only one format should be used. For example: There may be TimeSeriesWide and NumericLong, but there should _not_ be TimeSeriesWide and TimeSeriesLong.
* The borders between the types are derived from adjacent frames (within the array of frames) that share the same data type.
* If a reader does not opt-in into multi-type responses, it should be able to get the first data type that matches what the reader is looking for.

## Versioning

An object of this contract is to be as stable as possible. When `1.0` is reached, changes should be limited to enhancements until a `2.0` version, which is something that should generally be avoided in datatype versions.

### Contract Versioning

The contract version is only for concepts impacting the overarching concepts of the contract such as error handling or multi data type responses.

The addition of new datatypes, or changing of datatypes does not impact the contract version.

### Datatype Versions

Each individual datatype is given a version in major/minor form (x.x). The version is located in the `Frame.Meta.TypeVersion` property.

* Version `0.0` means the datatype is either pre contract, or in very early development.
* Version `0.x` means the datatype is well defined in the contract, but may change based on things learned for wider usage.
* Version `1.0` should be a stable data type, and should have no changes from the previous version
* Minor version changes beyond `1.0` must be backward compatible for data reader implementations. They also must be forward compatible with other `1.x` versions for readers as well (but without enhancement support).

## Considerations to Add / Todo

* Meta-data (Frame and Field)
* If the type/schema is declared, do we need to support the case where, for whatever reason, the type can be considered multiple Kinds at once?
* So far ordering is ignored (For example, the order of Value Fields in TimeSeriesWide or the order of Frames in TimeSeriesMulti). Need to decide if ordering as any symantec meaning, if so what it is, and consider it properties of converting between formats
  * Note: Issue on ordering [https://github.com/grafana/grafana-plugin-sdk-go/issues/366](https://github.com/grafana/grafana-plugin-sdk-go/issues/366) , not sure if it is display issue or not at this time

<!-- Footnotes themselves at the bottom. -->
## Notes

[^1]:

     In theory they can still be passed for things like visualization because Fields do have a numeric ordering within the frame, but this won't work with things like SSE/alerting.

[^2]:

     Using Field Name keeps naming consistent with the TimeSeriesMulti format (vs using the Frame Name)
