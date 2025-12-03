# Build Instructions

## Prerequisites

- Go 1.21 or later

## Building the Project

```bash
# Navigate to project directory
cd e:\go\dnp3-go

# Build all packages
go build ./...

# Run tests (when available)
go test ./...
```

## Building Examples

```bash
# Build master example
go build -o bin/simple_example.exe examples/simple_example.go

# Build custom channel example
go build -o bin/mock_channel_example.exe examples/custom_channel/mock_channel.go
```

## Common Issues

### Circular Import Error

If you see circular import errors, make sure you're on the latest version where config types are separated.

### Module Path

The module path is `avaneesh/dnp3-go`. If you want to use a different module path:

1. Update `go.mod`
2. Run `find . -name "*.go" -exec sed -i 's/avaneesh\/dnp3-go/your-module-path/g' {} \;`

## Development

### Adding New Transport

Implement the `PhysicalChannel` interface in `pkg/channel/interface.go`:

```go
type PhysicalChannel interface {
    Read(ctx context.Context) ([]byte, error)
    Write(ctx context.Context, data []byte) error
    Close() error
    Statistics() TransportStats
}
```

See `examples/custom_channel/mock_channel.go` for an example.
