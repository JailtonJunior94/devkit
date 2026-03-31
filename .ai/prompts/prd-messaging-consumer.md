# Prompt: Criar PRD - Messaging Consumer Agonístico e de Alta Performance

Este prompt deve ser utilizado para iniciar a criação de um PRD (Product Requirements Document) para um módulo de consumo de mensagens (Consumer) que seja agnóstico à implementação (Kafka, RabbitMQ, SQS, etc.), priorizando performance extrema, segurança e extensibilidade.

---

## Objetivo
Criar um PRD para um componente de **Messaging Consumer** que permita:
- Registro de handlers para diferentes tipos de eventos.
- Implementação inicial plugável para **Kafka** (utilizando `github.com/segmentio/kafka-go`).
- Design extensível para futuras implementações (RabbitMQ, Azure Service Bus, SQS).
- Garantia de robustez: sem `nil pointer`, sem `race conditions`, sem `memory leaks`.
- Aplicação de padrões de projeto (ex: Strategy, Factory, Adapter) baseados em [Refactoring Guru](https://refactoring.guru/design-patterns).

## Base de Código (Contratos)
O sistema deve ser projetado em torno das seguintes definições:

```go
package consumer

import (
	"context"
)

type ConsumerOptions func(consumer *consumer)
type ConsumeHandler func(ctx context.Context, params map[string]string, body []byte) error

type Consumer interface {
	// Consume inicia o loop de consumo de mensagens. Deve ser bloqueante ou rodar em background conforme a implementação.
	Consume(ctx context.Context) error
	
	// RegisterHandler mapeia um tipo de evento para um processador específico.
	RegisterHandler(eventType string, handler ConsumeHandler)
	
	// Shutdown garante o fechamento gracioso das conexões e interrupção do consumo.
	Shutdown(ctx context.Context) error
}
```

## Diretrizes para o PRD (Template: .claude/templates/prd-template.md)

### 1. Visão Geral
O Consumer deve abstrair a complexidade de brokers de mensagens, permitindo que a aplicação foque no processamento da regra de negócio através de handlers registrados, independente se a origem é Kafka ou outra ferramenta.

### 2. Objetivos e Sucesso
- **Abstração:** Interface única para múltiplos brokers.
- **Kafka-First:** Implementação robusta para Kafka como prova de conceito inicial.
- **Graceful Shutdown:** Garantir que nenhuma mensagem em processamento seja perdida ou deixada em estado inconsistente ao desligar.
- **Concorrência Segura:** Gerenciamento de workers e goroutines sem race conditions.
- **Zero Leaks:** Fechamento correto de conexões e release de recursos.

### 3. Funcionalidades Core
- Inicialização de consumidores via Factory ou Builder.
- Loop de consumo resiliente (retry logic, reconnection).
- Despacho de mensagens para handlers baseados no `eventType`.
- Suporte a metadados/headers via `params map[string]string`.
- Shutdown cooperativo via `context.Context`.

### 4. Restrições Técnicas de Alto Nível
- **Kafka Library:** `github.com/segmentio/kafka-go`.
- **Patterns:** Utilizar padrões para isolar a lógica do broker (Strategy/Adapter).
- **Performance:** Uso eficiente de buffers e processamento concorrente.
- **Safety:** Verificações rigorosas de ponteiros e estados de conexão.

### 5. Fora de Escopo
- Implementações de RabbitMQ, SQS ou Service Bus (serão tarefas futuras).
- Lógica de Dead Letter Queue (DLQ) complexa (deve ser tratada em PRD específico de resiliência).

---

**Instrução para o Agente:**
Gere o PRD seguindo rigorosamente o template em `.claude/templates/prd-template.md`. Foque em detalhar requisitos não funcionais de performance, extensibilidade e a natureza agnóstica do design.
