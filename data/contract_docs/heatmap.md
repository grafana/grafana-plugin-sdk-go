# Heatmap 

Status: EARLY Draft/Proposal

Heatmaps are used to show the magnitude of a phenomenon as color in two dimensions. The variation in color 
may give visual cues about how the phenomenon is clustered or varies over space. 


## Heatmap buckets (HeatmapBuckets)

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
    <td>1653416391000</td>
    <td>6</td>
    <td>7</td>
    <td>8</td>
  </tr>
  <tr>
    <td>1653416391000</td>
    <td>6</td>
    <td>7</td>
    <td>8</td>
  </tr>
  <tr>
    <td>1653416391000</td>
    <td>6</td>
    <td>7</td>
    <td>8</td>
  </tr>
</table>


## Heatmap scanlines (HeatmapScanlines)

In this format, each row in the frame indicates the value of a single cell in a heatmap.
There exists a row for every cell in the heatmap.

**Example:**

<table>
  <tr>
    <td>
      <strong>Type: Time</strong>
      <p>
        <strong>Name: xMax</strong>
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
    <td>1653416391000</td>
    <td>100</td>
    <td>1</td>
  </tr>
  <tr>
    <td>1653416391000</td>
    <td>200</td>
    <td>2</td>
  </tr>
  <tr>
    <td>1653416391000</td>
    <td>300</td>
    <td>3</td>
  </tr>

  <tr>
    <td>1653416392000</td>
    <td>100</td>
    <td>4</td>
  </tr>
  <tr>
    <td>1653416392000</td>
    <td>200</td>
    <td>5</td>
  </tr>
  <tr>
    <td>1653416392000</td>
    <td>300</td>
    <td>6</td>
  </tr>
</table>

This format requires uniform cell sizing


## Heatmap sparse (HeatmapSparse)

This format is simplar to Heatmap scanlines, except that each cell is independent from its adjacent values.
Unlike scanlines, this allows resolutions to change over time.  When 

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
    <td>1653416391000</td>
    <td>1653416392000</td>
    <td>100</td>
    <td>200</td>
    <td>1</td>
  </tr>
  <tr>
    <td>1653416392000</td>
    <td>1653416393000</td>
    <td>200</td>
    <td>400</td>
    <td>2</td>
  </tr>
</table>

* For high resolution with many gaps, this will require less space
* This format is much less optomized for fast render and lookup than the uniform "scanlines" approach
