# DevKit Observability (o11y)

[![Go Reference](https://pkg.go.dev/badge/devkit/o11y.svg)](https://pkg.go.dev/devkit/o11y)
[![Go Report Card](https://goreportcard.com/badge/github.com/jailtonjunior/devkit)](https://goreportcard.com/report/github.com/jailtonjunior/devkit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

O **DevKit Observability (`o11y`)** é uma biblioteca Go de padrão enterprise projetada para simplificar drasticamente a implementação do **OpenTelemetry (OTel)** em microserviços. 

Em vez de lidar com a complexidade de configurar múltiplos providers, exporters e propagadores manualmente, o `o11y` oferece uma fachada unificada e opinativa que garante consistência em **Tracing, Metrics e Logging** com o mínimo de esforço.

---

## 📑 Sumário

- [Destaques](#-destaques)
- [Por que usar o DevKit?](#-por-que-usar-o-devkit)
- [Instalação](#-instalação)
- [Início Rápido](#-início-rápido)
- [Guia de Utilização](#-guia-de-utilização)
    - [Rastreamento (Tracing)](#rastreamento-tracing)
    - [Métricas (Metrics)](#métricas-metrics)
    - [Logs Estruturados (Logging)](#logs-estruturados-logging)
- [Configuração e Opções](#-configuração-e-opções)
- [Testes e Mocking](#-testes-e-mocking)
- [Boas Práticas](#-boas-práticas)
- [Licença](#-licença)

---

## ✨ Destaques

- 🛡️ **Abstração Enterprise**: Esconde a verbosidade do SDK oficial do OpenTelemetry.
- 🚀 **Setup Unificado**: Inicialize Tracing, Metrics e Logging em uma única chamada.
- 📊 **Suporte OTLP Nativo**: Exportação simplificada via gRPC e HTTP (v1).
- 🪵 **Integração slog**: Bridge nativo para o `log/slog` do Go, correlacionando logs com traces.
- 🧪 **Testabilidade de Primeira Classe**: Pacote `oteltest` para validação de sinais em memória.
- 🔗 **Propagação de Contexto**: Suporte fácil para W3C TraceContext e Baggage.
- 🛑 **Graceful Shutdown**: Gerenciamento limpo de flush de dados e encerramento de conexões.

---

## 🤔 Por que usar o DevKit?

Configurar o OpenTelemetry corretamente envolve muitas decisões: qual sampler usar? Como configurar o resource? Como garantir que os logs tenham o `trace_id` correto? 

O `devkit/o11y` resolve isso ao:
1.  **Reduzir Boilerplate**: O que levaria ~100 linhas de código OTel puro é feito em ~10.
2.  **Garantir Consistência**: Todos os sinais (logs, métricas, traces) compartilham os mesmos atributos de recurso (nome do serviço, versão, ambiente).
3.  **Evitar Estado Global**: Por padrão, não registra providers globais, facilitando o isolamento em testes (exceto quando solicitado explicitamente para propagação).

---

## 📦 Instalação

```bash
go get devkit
```

---

## ⚡ Início Rápido

O exemplo abaixo mostra como configurar o SDK completo enviando dados para um coletor local.

```go
package main

import (
	"context"
	"log"

	"devkit/o11y"
	"devkit/o11y/otlpgrpc" // Recomendado para performance
)

func main() {
	ctx := context.Background()

	// 1. Inicialização Unificada
	sdk, err := o11y.New(ctx, o11y.Config{
		ServiceName:    "order-api",
		ServiceVersion: "1.0.0",
		Environment:    "production",
	}, 
	otlpgrpc.WithTrace(),  // Exporta traces via gRPC (default: localhost:4317)
	otlpgrpc.WithMetric(), // Exporta métricas via gRPC
	otlpgrpc.WithLog(),    // Exporta logs via gRPC
	o11y.WithW3CPropagators(), // Habilita propagação distribuída
	)
	if err != nil {
		log.Fatalf("falha ao configurar o11y: %v", err)
	}
	
	// 2. Garante o flush dos dados no encerramento
	defer sdk.Shutdown(ctx)

	// 3. Utilização
	logger := sdk.Logger()
	logger.Info("Aplicação iniciada com observabilidade completa!")
}
```

---

## 🔍 Guia de Utilização

### Rastreamento (Tracing)

O rastreamento permite visualizar o fluxo de uma requisição.

```go
func processOrder(ctx context.Context, orderID string) {
    tracer := sdk.TracerProvider().Tracer("orders")
    
    // Inicia um span
    ctx, span := tracer.Start(ctx, "process-order")
    defer span.End()

    span.SetAttributes(attribute.String("order.id", orderID))

    // O logger injetará automaticamente o trace_id no log
    sdk.Logger().InfoContext(ctx, "validando estoque")
}
```

### Métricas (Metrics)

Capture dados quantitativos sobre o comportamento do sistema.

```go
import "go.opentelemetry.io/otel/metric"

func recordPayment(sdk *o11y.Observability) {
    meter := sdk.MeterProvider().Meter("payments")
    
    counter, _ := meter.Int64Counter("payments_total", 
        metric.WithDescription("Total de pagamentos processados"),
    )

    counter.Add(context.Background(), 1, 
        metric.WithAttributes(attribute.String("status", "success")),
    )
}
```

### Logs Estruturados (Logging)

A integração com `slog` garante que seus logs sejam estruturados e compatíveis com o padrão OTLP.

```go
logger := sdk.Logger()

// Log simples
logger.Info("usuário logado", "user_id", 42)

// Log com contexto (inclui TraceID/SpanID se houver um span ativo)
logger.ErrorContext(ctx, "falha na conexão com banco", 
    "db_host", "localhost",
    "error", err,
)
```

---

## ⚙️ Configuração e Opções

A função `o11y.New` aceita a struct `Config` e múltiplas `Options`.

### Config Struct

| Atributo | Descrição |
| :--- | :--- |
| `ServiceName` | **(Obrigatório)** Identificador do serviço. |
| `ServiceVersion` | Versão da aplicação (ex: tag git ou semver). |
| `Environment` | Ambiente (ex: "prod", "dev", "staging"). |
| `ResourceAttributes` | Lista de `attribute.KeyValue` extras para o recurso. |

### Options Disponíveis

| Opção | Descrição |
| :--- | :--- |
| `otlpgrpc.WithTrace(endpoint)` | Configura exporter de Trace via gRPC. |
| `otlpgrpc.WithMetric(endpoint)`| Configura exporter de Metrics via gRPC. |
| `otlpgrpc.WithLog(endpoint)`   | Configura exporter de Logs via gRPC. |
| `otlphttp.WithTrace(endpoint)` | Configura exporter de Trace via HTTP. |
| `otlphttp.WithMetric(endpoint)`| Configura exporter de Metrics via HTTP. |
| `otlphttp.WithLog(endpoint)`   | Configura exporter de Logs via HTTP. |
| `WithSampler(sampler)`         | Define estratégia de amostragem (AlwaysOn, AlwaysOff, ParentBased, etc). |
| `WithMetricInterval(duration)` | Intervalo entre exportações de métricas (Default: 60s). |
| `WithW3CPropagators()`         | Ativa propagação de contexto (TraceContext + Baggage). |

---

## 🧪 Testes e Mocking

Não é necessário rodar um coletor OTel para seus testes unitários. Use o `oteltest`.

```go
import (
    "devkit/o11y/oteltest"
    "testing"
)

func TestBusinessLogic(t *testing.T) {
    // Cria um tracer em memória
    fake := oteltest.NewFakeTracer()
    
    // Execute sua lógica passando o provider fake
    tracer := fake.Tracer("test")
    _, span := tracer.Start(context.Background(), "op")
    span.End()

    // Verifique os spans gerados
    spans := fake.Spans()
    if len(spans) != 1 {
        t.Errorf("esperava 1 span, obteve %d", len(spans))
    }
}
```

---

## 💡 Boas Práticas

1.  **Singleton**: Inicialize o `Observability` uma única vez no `main.go` e compartilhe o objeto ou seus providers.
2.  **Context Everywhere**: Sempre passe o `context.Context` em suas funções para garantir que o rastreamento e a correlação de logs funcionem corretamente.
3.  **Defer Shutdown**: Sempre utilize `defer sdk.Shutdown(ctx)` logo após a inicialização para evitar perda de dados em buffering.
4.  **Use gRPC em Prod**: Para ambientes de alta performance, prefira os exporters gRPC (`otlpgrpc`).

---

## 🤝 Contribuição

Contribuições são bem-vindas! Se você encontrar um bug ou tiver uma sugestão de melhoria, sinta-se à vontade para abrir uma issue ou enviar um PR.

---

## 📄 Licença

Distribuído sob a licença MIT. Veja `LICENSE` para mais informações.
