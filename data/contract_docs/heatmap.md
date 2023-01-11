# Heatmap

Status: EARLY Draft/Proposal

Heatmaps are used to show the magnitude of a phenomenon as color in two dimensions. The variation in color may give visual cues about how the phenomenon is clustered or varies over space.

## Heatmap buckets (HeatmapBuckets)

Version: 0.0

The first field represents the X axis, the rest of the fields indicate rows in the heatmap.  
The true numeric range of each bucket can be indicated using an "le" label.  When absent,
The field display is used for the bucket label.

Example:

<table>
  <tr>
    <td>
      <strong>Type: Time</strong>
      <p>
        <strong>Name: Time</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: </strong>
      </p>
      <p>
        <strong>Labels: {"le":<em> "10"</em>}</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: </strong>
      </p>
      <p>
        <strong>Labels: {"le":<em> "20"</em>}</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: </strong>
      </p>
      <p>
        <strong>Labels: {"le":<em> "+Inf"</em>}</strong>
      </p>
    </td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:51</td>
    <td>6</td>
    <td>7</td>
    <td>8</td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:51</td>
    <td>6</td>
    <td>7</td>
    <td>8</td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:51</td>
    <td>6</td>
    <td>7</td>
    <td>8</td>
  </tr>
</table>

Note: [Timeseries wide](./timeseries.md#time-series-wide-format-timeserieswide) can be used directly
as heatmap-buckets, in this case each value field becomes a row in the heatmap.

## Heatmap scanlines (HeatmapScanlines)

Version: 0.0

In this format, each row in the frame indicates the value of a single cell in a heatmap.
There exists a row for every cell in the heatmap.

**Example:**

<table>
  <tr>
    <td>
      <strong>Type: Time</strong>
      <p>
        <strong>Name: xMax|xMin|x</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: yMax|yMin|y</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: Count</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: Total</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: Speed</strong>
      </p>
    </td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:51</td>
    <td>100</td>
    <td>1</td>
    <td>1</td>
    <td>1</td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:51</td>
    <td>200</td>
    <td>2</td>
    <td>2</td>
    <td>2</td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:51</td>
    <td>300</td>
    <td>3</td>
    <td>3</td>
    <td>3</td>
  </tr>

  <tr>
    <td>2022-05-24 18:19:52</td>
    <td>100</td>
    <td>4</td>
    <td>4</td>
    <td>4</td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:52</td>
    <td>200</td>
    <td>5</td>
    <td>5</td>
    <td>5</td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:52</td>
    <td>300</td>
    <td>6</td>
    <td>6</td>
    <td>6</td>
  </tr>
</table>

This format requires uniform cell sizing. The size of the cell is defined by the columns in each row that are chosen as the xMax|xMin|x and the yMax|yMin|y. We can see that the Number column(yMax|yMin|y) increases by 100(each cell is roughly 100 higher than the previous cell on the y axis) for each row containing a similar Time value(these stacked cells all have roughly the same location along the x axis). This produces a uniform cell size.

Note that multiple "value" fields can included to represent multiple dimensions within the same cell.  
The first value field is used in the display, unless explicilty configured

The field names for yMax|yMin|y indicate the aggregation period or the supplied values.

* yMax: the values are from the bucket below
* yMin: the values are from to bucket above
* y: the values are in the middle of the bucket

## Heatmap sparse (HeatmapSparse)

Version: 0.0

This format is simplar to Heatmap scanlines, except that each cell is independent from its adjacent values.
Unlike scanlines, this allows resolutions to change over time. Where scanline has uniformity of cells over time, heatmap sparse allows for variability of cells along the x axis(Time).

Example:

<table>
  <tr>
    <td>
      <strong>Type: Time</strong>
      <p>
        <strong>Name: xMin</strong>
      </p>
    </td>
    <td>
      <strong>Type: Time</strong>
      <p>
        <strong>Name: xMax</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: yMin</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: yMax</strong>
      </p>
    </td>
    <td>
      <strong>Type: Number</strong>
      <p>
        <strong>Name: Value</strong>
      </p>
    </td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:51</td>
    <td>2022-05-24 18:19:52</td>
    <td>100</td>
    <td>200</td>
    <td>1</td>
  </tr>
  <tr>
    <td>2022-05-24 18:19:52</td>
    <td>2022-05-24 18:19:53</td>
    <td>200</td>
    <td>400</td>
    <td>2</td>
  </tr>
</table>

* For high resolution with many gaps, this will require less space
* This format is much less optomized for fast render and lookup than the uniform "scanlines" approach
