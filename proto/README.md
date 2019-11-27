
Repository Structure?
*best:*
everything in grafana repo
*alternative:*
if getting go to play nice with that structure is super hard, lets create two repos:


## grafana-backend-platform
This will include all protobuf specs, and the core packages that will be used
in *both* plugins and in grafana core
```
 /proto
 /scripts
 /genproto // created
 /testdata // saved arrow files and maybe some CSV/json?
 /go/dataframe/...  // Used in grafana core and plugins-sdk
 /README
```

## grafana-plugin-sdk
This repo should hold a plugin-sdk for each of the languages we will support
For now, that is just `go`.  Lets take what we have, move parts to `grafana-backend-platform`
and the rest into a subfolder
```
 /README.md
 /go/...
```

## grafana
grafana core will import from `grafana-backend-platform`


------
------



