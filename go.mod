module github.com/grafana/grafana-plugin-sdk-go

go 1.23.5

require (
	github.com/apache/arrow-go/v18 v18.0.1-0.20241212180703-82be143d7c30
	github.com/cheekybits/genny v1.0.0
	github.com/chromedp/cdproto v0.0.0-20220208224320-6efb837e6bc2
	github.com/elazarl/goproxy v1.7.0
	github.com/getkin/kin-openapi v0.129.0
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/grafana/otel-profiling-go v0.5.1
	github.com/grafana/pyroscope-go/godeltaprof v0.1.8
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.0.1
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.2.0
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.6.3
	github.com/invopop/jsonschema v0.13.0 // for schema codgen+extraction
	github.com/json-iterator/go v1.1.12
	github.com/jszwedko/go-datemath v0.1.1-0.20230526204004-640a500621d6
	github.com/magefile/mage v1.15.0
	github.com/mattetti/filebuffer v1.0.1
	github.com/mitchellh/reflectwalk v1.0.2
	github.com/olekukonko/tablewriter v0.0.5
	github.com/prometheus/client_golang v1.21.0
	github.com/prometheus/common v0.62.0
	github.com/stretchr/testify v1.10.0
	github.com/unknwon/bra v0.0.0-20200517080246-1e3013ecaff8
	github.com/urfave/cli v1.22.16
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.59.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.59.0
	go.opentelemetry.io/contrib/propagators/jaeger v1.34.0
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.28.0
	go.opentelemetry.io/otel v1.34.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.34.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.34.0
	go.opentelemetry.io/otel/sdk v1.34.0
	go.opentelemetry.io/otel/trace v1.34.0
	golang.org/x/exp v0.0.0-20240909161429-701f63a606c0
	golang.org/x/net v0.35.0
	golang.org/x/oauth2 v0.26.0
	golang.org/x/sync v0.11.0
	golang.org/x/sys v0.30.0
	golang.org/x/text v0.22.0
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.5
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f // @grafana/grafana-app-platform-squad
)

require google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f

require (
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.25.1 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oasdiff/yaml v0.0.0-20241210131133-6b86fb107d80 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20241210130736-a94c01f36349 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/unknwon/com v1.0.1 // indirect
	github.com/unknwon/log v0.0.0-20150304194804-e617c87089d3 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250115164207-1a7da9e5054f // indirect
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8 // indirect
)
