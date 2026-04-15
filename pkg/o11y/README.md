# `pkg/o11y`

Facade de observabilidade para Go com tracing, logging estruturado e métricas sem espalhar dependências do OpenTelemetry pelas camadas de negócio.

Este pacote faz parte do módulo [`devkit`](https://github.com/jailtonjunior/devkit) e organiza a observabilidade em três níveis:

- `pkg/o11y`: contratos estáveis para a aplicação consumir
- `pkg/o11y/noop` e `pkg/o11y/fake`: implementações para desligar telemetria ou testar em memória
- `pkg/o11y/otel`, `pkg/o11y/tracing`, `pkg/o11y/metrics`, `pkg/o11y/logging`: integrações e bootstraps baseados em OpenTelemetry

## Instalação

No módulo `devkit`:

```bash
go get devkit
```

Para validar o pacote localmente:

```bash
go test ./pkg/o11y/...
```

## Quando usar

Use `pkg/o11y` quando você precisa:

- instrumentar código de negócio sem importar `go.opentelemetry.io/otel` fora da infraestrutura
- injetar uma dependência única para tracing, logs e métricas
- alternar entre `noop`, `fake` e `otel` sem mudar assinaturas de serviços e handlers
- manter correlação de logs com `trace_id` e `span_id` quando houver span ativo

Não é responsabilidade deste pacote:

- registrar providers globais do OpenTelemetry
- instrumentar automaticamente HTTP, SQL, gRPC ou Kafka
- subir um collector embutido
- fornecer middleware pronto para frameworks

## Uso Rápido

O ponto de consumo da aplicação é a interface `o11y.Signals`:

```go
type Signals interface {
    Tracer() Tracer
    Logger() Logger
    Metrics() Metrics
}
```

Exemplo de uso em uma camada de aplicação:

```go
package orders

import (
    "context"

    "devkit/pkg/o11y"
)

type Service struct {
    obs o11y.Signals
}

func NewService(obs o11y.Signals) *Service {
    return &Service{obs: obs}
}

func (s *Service) Create(ctx context.Context, customerID string) error {
    ctx, span := s.obs.Tracer().Start(
        ctx,
        "orders.Service.Create",
        o11y.WithSpanKind(o11y.SpanKindInternal),
        o11y.WithAttributes(o11y.String("customer.id", customerID)),
    )
    defer span.End()

    s.obs.Logger().Info(ctx, "creating order", o11y.String("customer.id", customerID))

    requests, err := s.obs.Metrics().Counter(
        "orders.create.requests",
        "Total order creation attempts",
        "1",
    )
    if err != nil {
        return err
    }
    requests.Increment(ctx, o11y.String("customer.id", customerID))

    return nil
}
```

## Providers Disponíveis

| Provider | Pacote | Quando usar |
|---------|--------|-------------|
| No-op | `devkit/pkg/o11y/noop` | desligar observabilidade com a mesma API |
| Fake | `devkit/pkg/o11y/fake` | testes unitários com assertions sobre spans, logs e métricas |
| OpenTelemetry | `devkit/pkg/o11y/otel` | produção ou integração real com exporters OTLP |

### `noop`

```go
import "devkit/pkg/o11y/noop"

obs := noop.NewProvider()
```

### `fake`

```go
import "devkit/pkg/o11y/fake"

obs := fake.NewProvider()
```

### `otel`

`otel.NewProvider` monta tracing, métricas e logs a partir de uma configuração única. Cada exporter é opcional; quando um exporter é `nil`, aquele sinal cai para no-op sem quebrar a API.

```go
package main

import (
    "context"
    "log"

    "devkit/pkg/o11y"
    "devkit/pkg/o11y/otel"
)

func main() {
    ctx := context.Background()

    obs, err := otel.NewProvider(ctx, &otel.Config{
        ServiceName:    "checkout",
        ServiceVersion: "1.3.0",
        Environment:    "development",
        LogLevel:       o11y.LogLevelDebug,
        LogFormat:      o11y.LogFormatText,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer func() { _ = obs.Shutdown(ctx) }()

    obs.Logger().Info(ctx, "observability initialized")
}
```

## API Pública

### Campos estruturados

Os atributos compartilhados por logs, spans e métricas usam `o11y.Field`:

```go
o11y.String("service", "billing")
o11y.Int("attempt", 3)
o11y.Int64("duration_ms", 42)
o11y.Float64("amount", 149.90)
o11y.Bool("sampled", true)
o11y.Error(err)
o11y.Any("payload", map[string]any{"id": "123"})
```

### Tracing

O `Tracer` expõe:

- `Start(ctx, name, opts...)`
- `SpanFromContext(ctx)`
- `ContextWithSpan(ctx, span)`

O `Span` expõe:

- `End()`
- `SetAttributes(fields...)`
- `SetStatus(code, description)`
- `RecordError(err, fields...)`
- `AddEvent(name, fields...)`
- `Context()`

Opções de criação de span:

- `o11y.WithSpanKind(...)`
- `o11y.WithAttributes(...)`

Valores suportados:

- `o11y.SpanKindInternal`
- `o11y.SpanKindServer`
- `o11y.SpanKindClient`
- `o11y.SpanKindProducer`
- `o11y.SpanKindConsumer`
- `o11y.StatusCodeUnset`
- `o11y.StatusCodeOK`
- `o11y.StatusCodeError`

### Logging

O `Logger` expõe:

- `Debug`
- `Info`
- `Warn`
- `Error`
- `With(fields...)`

Formatos suportados:

- `o11y.LogFormatText`
- `o11y.LogFormatJSON`

Níveis suportados:

- `o11y.LogLevelDebug`
- `o11y.LogLevelInfo`
- `o11y.LogLevelWarn`
- `o11y.LogLevelError`

Na implementação `otel`, o logger escreve via `slog` e também tenta emitir logs OTLP quando `LogExporter` está configurado.

### Métricas

O `Metrics` expõe:

- `Counter(name, description, unit string)`
- `Histogram(name, description, unit string)`
- `UpDownCounter(name, description, unit string)`
- `Gauge(name, description, unit string, callback)`

Comportamento dos instrumentos:

- `Counter` aceita incrementos positivos e expõe `Add` e `Increment`
- `Histogram` registra distribuições com `Record`
- `UpDownCounter` aceita deltas positivos e negativos
- `Gauge` é assíncrono e retorna `o11y.ErrNilGaugeCallback` se o callback for `nil`

## Arquitetura dos Subpacotes

| Caminho | Papel |
|---------|-------|
| `pkg/o11y/o11y.go` | contratos centrais (`Signals`, `Field`, `Span`, `SpanOption`) |
| `pkg/o11y/tracer.go` | interface de tracing |
| `pkg/o11y/logger.go` | interface de logging estruturado |
| `pkg/o11y/metrics.go` | interface de métricas |
| `pkg/o11y/noop/` | implementação no-op |
| `pkg/o11y/fake/` | doubles em memória para testes |
| `pkg/o11y/otel/` | facade compatível com OpenTelemetry |
| `pkg/o11y/tracing/` | bootstrap isolado de `TracerProvider` |
| `pkg/o11y/metrics/` | bootstrap isolado de `MeterProvider` |
| `pkg/o11y/logging/` | bootstrap isolado de `slog.Logger` com bridge OTel |
| `pkg/o11y/oteltest/` | fakes focados em testes de integrações OTel |

## Uso Por Sinal

Se você não precisa da facade unificada, os subpacotes abaixo expõem bootstraps independentes:

- `devkit/pkg/o11y/tracing`
- `devkit/pkg/o11y/metrics`
- `devkit/pkg/o11y/logging`

Exemplos mínimos:

```go
tracingProvider, err := tracing.New(ctx, tracing.Config{ServiceName: "checkout"})
metricsProvider, err := metrics.New(ctx, metrics.Config{ServiceName: "checkout"})
loggingProvider, err := logging.New(ctx, logging.Config{ServiceName: "checkout"})
```

Sem exporter configurado, esses providers continuam válidos, mas operam em modo no-op para o respectivo sinal.

## Testes

Para testes de unidade da sua aplicação:

- use `fake.NewProvider()` quando quiser inspecionar spans, logs e medições pela facade `o11y`
- use `oteltest` quando estiver testando componentes que dependem diretamente de tipos do ecossistema OpenTelemetry

Exemplo com `oteltest`:

```go
logger := oteltest.NewFakeLogger()
logger.Logger().Info("hello")
```

## Contribuição

Mudanças em `pkg/o11y` devem manter o contrato de aplicação desacoplado do SDK concreto. Antes de abrir PR:

```bash
go test ./pkg/o11y/...
```

Se a mudança alterar a API pública, atualize este README e os exemplos em `*_test.go`.

## Licença

MIT. Veja a licença do repositório em [`LICENSE`](../../LICENSE).
