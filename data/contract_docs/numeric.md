# Numeric Kind Formats

Numeric Kinds are generally similar to their corresponding time series type, except that their value is a single number, instead of a series of (time, numeric value). So the value of each metric is a single number like 1, 2.3, or NaN

This generally corresponds to a prometheus instant vector, or a SQL table with string and number columns and multiple rows.

## Numeric Wide Format (NumericWide)

Version: 0.1

Example:

<table>
  <tr>
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
   <td>1
   </td>
   <td>6
   </td>
  </tr>
</table>

Properties:

* There should only be one frame with the type indicator
* There should be no rows or a single row in the frame
* All fields should have a numeric or bool type (e.g. if Go float64, *int, etc)
* Field Labels are used

Remainder Data:

* Any additional frames without the type indicator or a different one
* Any time or string fields

## Numeric Multi Format (NumericMulti)

Version: 0.1

This logically is no different than NumericWide, except that instead of having one frame with multiple Fields there are multiple frames with a single field.

**Example:**

Frame 0:

<table>
  <tr>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: {"host":<em> "a"}</em></strong>
   </td>
  </tr>
  <tr>
   <td>1
   </td>
  </tr>
</table>

Frame 1:


<table>
  <tr>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: {"host":<em> "b"</em>}</strong>
   </td>
  </tr>
  <tr>
   <td>6
   </td>
  </tr>
</table>

Properties:

* There should be no rows or a single row in the frame
* There should be one value field per frame

Remainder Data:

* Any time or string fields
* Any value fields after the first
* Any additional frames without the type indicator

## Numeric Long Format (NumericLong) [SQL-Table-Like]

Version: 0.1

This is the response one would imagine with a query like `Select Host, avg(cpu) … group by host". This is similar to the TimeSeriesLong format in that dimensions exist in string columns[^9].

Example:

<table>
  <tr>
   <td><strong>Type: Number</strong>
<p>
<strong>Name: cpu</strong>
<p>
<strong>Labels: nil</strong>
   </td>
   <td><strong>Type: String</strong>
<p>
<strong>Name: host</strong>
<p>
<strong>Labels: nil</strong>
   </td>
  </tr>
  <tr>
   <td>1
   </td>
   <td>a
   </td>
  </tr>
  <tr>
   <td>6
   </td>
   <td>b
   </td>
  </tr>
</table>


Properties:

* There should be a single Frame
* There may be one or more value fields 
* If there is more than one row there needs to be one or more string fields
* Each string column is a dimension, where the field/field name is the name of the dimension, and the corresponding values of the field are the dimensions value (e.g. a field with the name "host" would create a dimension like "host=web1" for a row/value in that field containing "web1"
* The Labels property of each Field is unused
* For each value field, the unique combination of item name (value Field Name) and its set of key (String field Name) and value (string field values) pairs form each unique item identifier.

Remainder Data

* Any additional frames with a different or no type indicator
* Any time fields

<!-- Footnotes themselves at the bottom. -->
## Notes

[^9]:
     Other than this connection, "Long" is perhaps a bad name in this context, numeric table perhaps?
