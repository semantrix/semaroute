# semaroute - High-Performance LLM Routing Gateway

A distributed, provider-agnostic LLM router with intelligent routing policies, failover, and comprehensive observability built in Go.

## 🚀 Features

- **Provider Agnostic**: Support for OpenAI, Anthropic, and extensible to other LLM providers
- **Intelligent Routing**: Cost-based and failover routing policies with automatic provider selection
- **High Performance**: Built with Go for high concurrency and low latency
- **Health Monitoring**: Continuous health checks with automatic failover
- **Observability**: Built-in structured logging, Prometheus metrics, and OpenTelemetry tracing
- **Caching**: In-memory caching for improved performance (Redis support planned)
- **Resilience**: Retry logic with exponential backoff and circuit breakers
- **RESTful API**: Clean HTTP API with comprehensive error handling

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │───▶│  semaroute      │───▶│  LLM Provider   │
│                 │    │  Gateway        │    │  (OpenAI, etc.) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  Routing        │
                       │  Policy Engine  │
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  Health         │
                       │  Checker        │
                       └─────────────────┘
```

### Core Components

- **Server**: HTTP server with middleware for observability
- **Router**: Intelligent routing based on policies (cost, failover)
- **Providers**: Abstracted LLM provider interfaces
- **Health Checker**: Continuous monitoring of provider health
- **Cache**: Response caching for improved performance
- **Observability**: Logging, metrics, and tracing

## 📦 Installation

### Prerequisites

- Go 1.21 or later
- Git

### Build from Source

```bash
# Clone the repository
git clone https://github.com/semantrix/semaroute.git
cd semaroute

# Build the binary
go build -o bin/semaroute-server ./cmd/semaroute-server

# Run the server
./bin/semaroute-server
```

### Using Go Modules

```bash
go mod tidy
go run ./cmd/semaroute-server
```

## ⚙️ Configuration

### Configuration File

Create a `config.yaml` file in your working directory:

```yaml
server:
  port: 8080

providers:
  openai:
    enabled: true
    api_key: "${OPENAI_API_KEY}"
    timeout: 30s

routing_policy:
  type: "cost_based"

observability:
  logging:
    level: "info"
  metrics:
    enabled: true
    port: 9090
```

### Environment Variables

```bash
export OPENAI_API_KEY="your-openai-api-key"
export ANTHROPIC_API_KEY="your-anthropic-api-key"
export SEMAROUTE_SERVER_PORT="8080"
```

### Command Line Options

```bash
./semaroute-server -config=config.yaml
./semaroute-server -version
```

## 🚀 Quick Start

1. **Set up API keys**:
   ```bash
   export OPENAI_API_KEY="your-key-here"
   export ANTHROPIC_API_KEY="your-key-here"
   ```

2. **Enable providers** in `config.yaml`:
   ```yaml
   providers:
     openai:
       enabled: true
     anthropic:
       enabled: true
   ```

3. **Start the server**:
   ```bash
   go run ./cmd/semaroute-server
   ```

4. **Test the API**:
   ```bash
   curl -X POST http://localhost:8080/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "gpt-4",
       "messages": [{"role": "user", "content": "Hello!"}]
     }'
   ```

## 📚 API Reference

### Chat Completions

```http
POST /v1/chat/completions
Content-Type: application/json

{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "temperature": 0.7,
  "max_tokens": 100
}
```

### Health Check

```http
GET /health
```

### Models

```http
GET /v1/models
```

### Metrics

```http
GET /metrics
```

## 🔧 Routing Policies

### Cost-Based Routing

Automatically selects the most cost-effective provider while considering latency and health:

```yaml
routing_policy:
  type: "cost_based"
  config:
    cost_weight: 0.6
    latency_weight: 0.3
    health_weight: 0.1
```

### Failover Routing

Primary/backup provider selection with automatic failover:

```yaml
routing_policy:
  type: "failover"
  config:
    primary_provider: "openai"
    backup_providers: ["anthropic"]
    failover_delay: 30s
```

## 📊 Monitoring

### Metrics

Prometheus metrics are available at `/metrics`:

- Request counts and durations
- Provider health and latency
- Routing decision metrics
- Cache performance

### Health Checks

```bash
# Overall health
curl http://localhost:8080/health

# Provider-specific health
curl http://localhost:8080/admin/providers/openai/health
```

### Logging

Structured JSON logging with configurable levels:

```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Provider request successful",
  "provider": "openai",
  "model": "gpt-4",
  "duration_ms": 1250
}
```

## 🧪 Development

### Project Structure

```
semaroute/
├── cmd/
│   └── semaroute-server/     # Main application entry point
├── internal/
│   ├── server/               # HTTP server and handlers
│   ├── router/               # Routing logic and policies
│   ├── providers/            # LLM provider implementations
│   ├── cache/                # Caching layer
│   ├── models/               # Data models
│   └── observability/        # Logging, metrics, tracing
├── pkg/
│   └── api/                  # Public API types
├── config.yaml               # Configuration file
└── go.mod                    # Go module file
```

### Running Tests

```bash
go test ./...
go test -v ./internal/...
```

### Building

```bash
# Development build
go build -o bin/semaroute-server ./cmd/semaroute-server

# Production build
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/semaroute-server ./cmd/semaroute-server
```

## 🔒 Security

- API keys are loaded from environment variables
- CORS is configurable for cross-origin requests
- Request validation and sanitization
- Rate limiting support (planned)

## 🚧 Roadmap

- [ ] Redis cache backend
- [ ] Additional LLM providers (Google, Cohere, etc.)
- [ ] Advanced routing policies (load balancing, A/B testing)
- [ ] Rate limiting and quota management
- [ ] WebSocket support for real-time streaming
- [ ] Kubernetes deployment manifests
- [ ] Docker containerization
- [ ] Performance benchmarking suite

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Add tests for new functionality
- Update documentation for API changes
- Use conventional commit messages

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Chi](https://github.com/go-chi/chi) for HTTP routing
- [Zap](https://github.com/uber-go/zap) for structured logging
- [Prometheus](https://prometheus.io/) for metrics
- [OpenTelemetry](https://opentelemetry.io/) for tracing
- [Viper](https://github.com/spf13/viper) for configuration management

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/semantrix/semaroute/issues)
- **Discussions**: [GitHub Discussions](https://github.com/semantrix/semaroute/discussions)
- **Documentation**: [GitHub Wiki](https://github.com/semantrix/semaroute/wiki)

---

**semaroute** - Intelligent LLM routing for the modern AI stack 🚀
