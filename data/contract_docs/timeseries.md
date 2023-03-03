# Time Series Kind Formats

## Properties Shared by All Time Series Based Formats

* Frames should be sorted by the time column/field in ascending order[^3]
* The Time field(s):
  * Should have no null values
  * Field name is for display purposes only, there should be no labels
* **_Value Field(s)_**
  * Value fields are called this because it is the field where the _value_ of each datapoint (time,value) is located.
  * It can be a numeric or bool field. For numeric values
    * Go: Float64, *Float64, or Int64 etc 
    * in JS 'number'
  * The series name comes the Value Field's Name property

### Invalid Cases

* The is not at least both a time field and a value field (unless the single frame "no data" case)
* The "No Data" case is present (a frame with no fields) alongside data
* Possibly Warning and not error:
  * Duplicate items (identified by name+dimensions)
  * Unsorted (time is not sorted from old to new)

## Time Series Wide Format (TimeSeriesWide)

Version: 0.1

The wide format has a set of time series in a single Frame that share the same time field. It is called "wide" because it gets _wider_ as more series are added.

**Example:**

<table>
  <tr>
   <td><strong>Type: Time</strong>
<p>
<strong>Name: T</strong>
<p>
<strong>Labels: nil</strong>
   </td>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: {"host":<em> "a"}</em></strong>
   </td>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: {"host":<em> "b"</em>}</strong>
   </td>
  </tr>
  <tr>
   <td>2022-04-27 5:00
   </td>
   <td>1
   </td>
   <td>6
   </td>
  </tr>
  <tr>
   <td>2022-04-27 6:00
   </td>
   <td>4
   </td>
   <td>8
   </td>
  </tr>
  <tr>
   <td>2022-04-27 7:00
   </td>
   <td>2
   </td>
   <td>5
   </td>
  </tr>
  <tr>
   <td>2022-04-27 8:00
   </td>
   <td>3
   </td>
   <td>9
   </td>
  </tr>
</table>

It should have the following properties: (Also see Shared Properties):

* The first field of type Time is the time index of all the time series.
* There should be only one Frame with the data type declaration.
* There should be at least one field that is a value Field Type 
* If there are multiple numeric fields, the combination of the time field with each value field in the frame creates each time series (metric)
* The time field should have no duplicate values (duplicate timestamps).

Remainder Data:

* Any additional Frames without the type declaration or a different declaration
* Any string fields in the Frame

Notes:

* A Go example of an approximation of this is [here](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/data#example-Frame-TSDBTimeSeriesSharedTimeIndex).

## Time Series Multi Format (TimeSeriesMulti)

Version: 0.1

The TimeSeriesMulti format has one time series per frame. If the response has multiple series where the time values may not line up, this format must be used over TimeSeriesWide.  The format is called "multi" because the data lives across _multiple_ data frames.

**Example**:

Frame 0:

<table>
  <tr>
   <td><strong>Type: Time</strong>
<p>
<strong>Name: T</strong>
<p>
<strong>Labels: nil</strong>
   </td>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: {"host": "a"}</strong>
   </td>
  </tr>
  <tr>
   <td>2022-04-27 5:00
   </td>
   <td>1
   </td>
  </tr>
  <tr>
   <td>2022-04-27 6:00
   </td>
   <td>4
   </td>
  </tr>
  <tr>
   <td>2022-04-27 7:00
   </td>
   <td>2
   </td>
  </tr>
  <tr>
   <td>2022-04-27 8:00
   </td>
   <td>3
   </td>
  </tr>
</table>

Frame 1:

<table>
  <tr>
   <td><strong>Type: Time</strong>
<p>
<strong>Name: T</strong>
<p>
<strong>Labels: nil</strong>
   </td>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: {"host": "b"}</strong>
   </td>
  </tr>
  <tr>
   <td>2022-04-27 5:00
   </td>
   <td>6
   </td>
  </tr>
  <tr>
   <td>2022-04-27 6:00
   </td>
   <td>8
   </td>
  </tr>
  <tr>
   <td>2022-04-27 7:00
   </td>
   <td>5
   </td>
  </tr>
  <tr>
   <td>2022-04-27 8:00
   </td>
   <td>9
   </td>
  </tr>
</table>

It should have the following properties: (Also see Shared Properties):

* Each frame should have at least time and one numeric value column. The first occurrence of each field of this type is used for the series.
* Different Frames can have different field lengths (but within a frame, they must be of the same length)
* Each time field should have no duplicate values (duplicate timestamps)

Remainder Data:

* Any numeric or time fields after the first of each in each frame
* Any additional Frames without the type declaration or a different declaration
* Any string fields in the Frame

Notes:

* Go example [here](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/data#example-Frame-TSDBTimeSeriesDifferentTimeIndices).
* The multi format is the only format that can be converted to from the other formats without data manipulation. Therefore it is a type that can contain the series information of all the other types.

## Time Series Long Format (TimeSeriesLong) [SQL-Like]

Version: 0.1

This is a response format common to SQL like systems[^4]. See [Grafana documentation: Multiple dimensions in table format](https://grafana.com/docs/grafana/latest/basics/timeseries-dimensions/#multiple-dimensions-in-table-format) for some more simple (but not complete) examples. It currently exists as a data transformation within some datasources[^5] in the backend that query SQL-like data, see [this Go Example for how that code works](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/data#example-Frame-TableLikeLongTimeSeries).

The format is called "Long" because there are more rows to hold the same series than the "wide" format and therefore it grows _longer_.

Example:

<table>
  <tr>
   <td><strong>Type: Time</strong>
<p>
<strong>Name: T</strong>
<p>
<strong>Labels: nil</strong>
   </td>
   <td><strong>Type: String</strong>
<p>
<strong>Name: host</strong>
<p>
<strong>Labels: nil</strong>
   </td>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: nil</strong>
   </td>
  </tr>
  <tr>
   <td>2022-04-27 5:00
   </td>
   <td>a
   </td>
   <td>1
   </td>
  </tr>
  <tr>
   <td>2022-04-27 5:00
   </td>
   <td>b
   </td>
   <td>6
   </td>
  </tr>
  <tr>
   <td>2022-04-27 6:00
   </td>
   <td>a
   </td>
   <td>4
   </td>
  </tr>
  <tr>
   <td>2022-04-27 6:00
   </td>
   <td>b
   </td>
   <td>8
   </td>
  </tr>
  <tr>
   <td>2022-04-27 7:00
   </td>
   <td>a
   </td>
   <td>2
   </td>
  </tr>
  <tr>
   <td>2022-04-27 7:00
   </td>
   <td>b
   </td>
   <td>5
   </td>
  </tr>
  <tr>
   <td>2022-04-27 8:00
   </td>
   <td>a
   </td>
   <td>3
   </td>
  </tr>
  <tr>
   <td>2022-04-27 8:00
   </td>
   <td>b
   </td>
   <td>9
   </td>
  </tr>
</table>

It should have the following properties: (Also see Shared Properties)::

* The first time field is used as the timestamps
* The Time field can have duplicate timestamps (but must be sorted in ascending time)
* There may optionally be string fields. For each string field:
  * The column/field Name is the dimension (e.g. "label") name
  * Corresponding string values in that field (by row) are the label values
* Series are constructed by iterating over the rows of the dataframe table response.
* The name of any value fields/columns becomes the name for each series
* The labels property of fields is not used

Remainder Data:

* Any additional time fields after the first
* Any additional Frames without the type declaration or a different declaration

Additional Properties or Considerations:

* In this format, the full dimension (e.g. "host"=value) is extracted from the values within a field, instead of being declared within the fields schema like the other formats.
* Since dimensions are represented in fields that are present for all derived series, this can not hold mixed dimension keys so all series will have the same set of dimension keys. For example, one could not have net.bytes{host="a"} and net.bytes{host="a",int="eth0"} together - the first would have to become net.bytes{host="a",**int=""**}
* It is unclear if a bool type Field should be considered a value field (e.g. and up/down metric) or a dimension (where it would be treated conceptually like labels) 

## Converting Between Time Series Formats

<table>
  <tr>
   <td><strong>Src</strong>
   </td>
   <td><strong>Dst</strong>
   </td>
   <td><strong>Modifies Data</strong>
   </td>
   <td><strong>Properties</strong>
   </td>
  </tr>
  <tr>
   <td>Wide
   </td>
   <td>Multi
   </td>
   <td><strong>No</strong>[^6]
   </td>
   <td>
<ul>

<li>One Frame to multiple Frames

<li>Each value (numeric) field alongside a copy of the time field from the Wide Frame is moved to its own individual Frame when converted to the Multi format
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Multi
   </td>
   <td>Wide
   </td>
   <td>Yes
   </td>
   <td>
<ul>

<li>Multiple Frames to one Frame

<li>A union must be performed: All of the Time fields from all the Frames in the Multi Format must become one Time Field for the Wide Frame

<li>Each value field is moved into the Wide Frame

<li>All value (numeric) Fields must be the same length as the time Field in the Wide Frame, therefore the value fields may need to be filled with zero values (probably null), effectively creating datapoints where none may have existed before
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Wide
   </td>
   <td><em>Long</em>[^7]
   </td>
   <td>Yes
   </td>
   <td>
<ul>

<li>One Frame to one Frame

<li>Labels are extracted from the value Fields, and become string Fields with a name that matches all the keys found in all labels. The label values become Field values in the corresponding fields.

<li>Because the string field will be present, label/dimension keys that exist for one series must exist for all the series, and therefore the series may be altered in that label keys that did not exist are created (likely with a value of null). This effectively may create series that didn't exist
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Long
   </td>
   <td>Wide
   </td>
   <td>Yes[^8]
   </td>
   <td>
<ul>

<li>One Frame to One Frame

<li>String Fields become labels on the value (numeric) fields (label keys come from Field name, label values from the Field's values.

<li>Repeated timestamps in a single time field from the Long frame are de-duplicated in the time field in the Wide Frame

<li>Because rows (timestamps) may be missing in the data of Long format, nulls may need to be inserted into the series to make them all share the same time field (a property of Wide)
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Long
   </td>
   <td>Multi
   </td>
   <td><strong>No</strong>
   </td>
   <td>
<ul>

<li>One Frame to Multiple Frames (with a series per frame)

<li>Each Frames Time field is built when a series is matched for that time
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Multi
   </td>
   <td><em>Long</em>
   </td>
   <td>Yes
   </td>
   <td>
<ul>

<li>Labels from the Multi format Frames become string Fields in the Long Frame, and the string columns are present for all rows in Frame for the Long format. Therefore label keys may get added to series
</li>
</ul>
   </td>
  </tr>
</table>

<!-- Footnotes themselves at the bottom. -->
## Notes

[^3]:

     This is because sorting is generally expensive in terms of resources, and is best done by the database behind a datasource in most cases.

[^4]:
     I don't believe our current SQL datasources strictly follow this, but some Azure ones do. This was either due to miscommunication about the intent of this format and the upgrade to Grafana 8 and/or lack of understanding about breaking changes, or both.

[^5]:
     This transformation happens when queried with "Format As=Time Series". The problem with the transformation happening at this stage of the pipeline is that while it does give the user Time Series for a common Time Series in Table format, it makes it so the "Table View" of the data doesn't like up with SQL returns from their query. **TODO: Define this general concept later, maybe call it "What you see is _NOT_ what you get", "Data Miscommunication", something.** This means we either need to return two things (sort of like exemplars?), or the operation should be moved, or something else.

[^6]:
<p>
     Of the time series format, only when the format being converted to is "Multi" can the underlying time series data not be manipulated

[^7]:
<p>
     In practice, I haven't seen any cases for converting to the Long format, only reading it in. Perhaps it may be requested as an export format at some point, but basically this is for illustration presently.

[^8]:
<p>
     This is used by the SQL datasources to extract time series from the "Long" format (via go sdk/data pkg). In hindsight I sort of wish we had gone with LongToMany instead. See "related" in <a href="https://github.com/grafana/grafana-plugin-sdk-go/issues/315#issuecomment-817839070">this issue comment</a>.