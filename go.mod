module github.com/mrz1836/go-cachestore

go 1.25.0

require (
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/coocood/freecache v1.2.7
	github.com/gomodule/redigo v1.9.3
	github.com/mrz1836/go-cache v1.1.10
	github.com/mrz1836/go-logger v1.0.6
	github.com/newrelic/go-agent/v3 v3.43.3
	github.com/pkg/errors v0.9.1
	github.com/rafaeljusto/redigomock v2.4.0+incompatible
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/yuin/gopher-lua v1.1.2 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.42.0
