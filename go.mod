module github.com/grafana/grafana-plugin-sdk-go

go 1.25.3

require (
	github.com/apache/arrow-go/v18 v18.4.1
	github.com/cheekybits/genny v1.0.0
	github.com/chromedp/cdproto v0.0.0-20250803210736-d308e07a266d
	github.com/elazarl/goproxy v1.7.2
	github.com/getkin/kin-openapi v0.133.0
	github.com/go-openapi/loads v0.23.2
	github.com/go-openapi/spec v0.22.1
	github.com/go-openapi/strfmt v0.25.0
	github.com/go-openapi/validate v0.25.1
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/grafana/otel-profiling-go v0.5.1
	github.com/grafana/pyroscope-go/godeltaprof v0.1.9
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.1.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.7.0
	github.com/invopop/jsonschema v0.13.0 // for schema codgen+extraction
	github.com/json-iterator/go v1.1.12
	github.com/jszwedko/go-datemath v0.1.1-0.20230526204004-640a500621d6
	github.com/magefile/mage v1.15.0
	github.com/mattetti/filebuffer v1.0.1
	github.com/mitchellh/reflectwalk v1.0.2
	github.com/olekukonko/tablewriter v0.0.5
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/common v0.67.4
	github.com/stretchr/testify v1.11.1
	github.com/unknwon/bra v0.0.0-20200517080246-1e3013ecaff8
	github.com/urfave/cli v1.22.17
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.63.0
	go.opentelemetry.io/contrib/propagators/jaeger v1.38.0
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.32.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0
	go.opentelemetry.io/otel/sdk v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	golang.org/x/exp v0.0.0-20251002181428-27f1f14c8bb9
	golang.org/x/net v0.47.0
	golang.org/x/oauth2 v0.33.0
	golang.org/x/sync v0.18.0
	golang.org/x/sys v0.38.0
	golang.org/x/text v0.31.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // @grafana/grafana-app-platform-squad
)

require github.com/go-openapi/swag/loading v0.25.3

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.24.1 // indirect
	github.com/go-openapi/errors v0.22.4 // indirect
	github.com/go-openapi/jsonpointer v0.22.1 // indirect
	github.com/go-openapi/jsonreference v0.21.3 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/swag/conv v0.25.3 // indirect
	github.com/go-openapi/swag/fileutils v0.25.1 // indirect
	github.com/go-openapi/swag/jsonname v0.25.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.3 // indirect
	github.com/go-openapi/swag/mangling v0.25.1 // indirect
	github.com/go-openapi/swag/stringutils v0.25.1 // indirect
	github.com/go-openapi/swag/typeutils v0.25.3 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.3 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/jaegertracing/jaeger-idl v0.5.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/unknwon/com v1.0.1 // indirect
	github.com/unknwon/log v0.0.0-20150304194804-e617c87089d3 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/woodsbury/decimal128 v1.3.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.mongodb.org/mongo-driver v1.17.6 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.1 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/telemetry v0.0.0-20251008203120-078029d740a8 // indirect
	golang.org/x/tools v0.38.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251022142026-3a174f9686a8 // indirect
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
)
