module github.com/mrz1836/go-cachestore

go 1.24.0

require (
	github.com/alicebob/miniredis/v2 v2.36.1
	github.com/coocood/freecache v1.2.5
	github.com/gomodule/redigo v1.9.3
	github.com/mrz1836/go-cache v1.0.7
	github.com/mrz1836/go-logger v1.0.4
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/pkg/errors v0.9.1
	github.com/rafaeljusto/redigomock v2.4.0+incompatible
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260203192932-546029d2fa20 // indirect
	google.golang.org/grpc v1.78.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.40.0
