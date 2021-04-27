module github.com/grafana/grafana-plugin-sdk-go

go 1.14

// At least version v1.3.0 creates issues with Grafana dependencies.
replace github.com/grpc-ecosystem/go-grpc-middleware => github.com/grpc-ecosystem/go-grpc-middleware v1.2.2

require (
	github.com/apache/arrow/go/arrow v0.0.0-20210223225224-5bea62493d91
	github.com/cheekybits/genny v1.0.0
	github.com/google/go-cmp v0.5.5
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.2.2
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/json-iterator/go v1.1.10
	github.com/magefile/mage v1.11.0
	github.com/mattetti/filebuffer v1.0.1
	github.com/mitchellh/reflectwalk v1.0.1
	github.com/olekukonko/tablewriter v0.0.5
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.21.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210423184538-5f58ad60dda6 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887
	google.golang.org/genproto v0.0.0-20210426193834-eac7f76ac494 // indirect
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)
