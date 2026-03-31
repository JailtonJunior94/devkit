# Prompt: Criar PRD - Event Dispatcher de Alta Performance

Este prompt deve ser utilizado para iniciar a criação de um PRD (Product Requirements Document) focado em um componente de Dispatcher de Eventos em Go, priorizando confiabilidade extrema, performance e segurança concorrente.

---

## Objetivo
Criar um PRD para um componente de Dispatcher de Eventos que seja:
- Extremamente confiável e performático.
- Livre de `panic` (resiliente a falhas de handlers).
- Livre de `memory leak` (gestão eficiente de referências).
- Livre de `race conditions` (thread-safe).

## Base de Código (Contratos)
O sistema deve evoluir a partir das seguintes interfaces:

```go
package events

import (
	"context"
)

type Event interface {
	GetEventType() string
	GetPayload() any
}

type EventDispatcher interface {
	Register(eventType string, handler EventHandler)
	Dispatch(ctx context.Context, event Event) error
	Remove(eventType string, handler EventHandler) error
	Has(eventType string, handler EventHandler) bool
	Clear()
}

type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}
```

## Diretrizes para o PRD (Template: .claude/templates/prd-template.md)

### 1. Visão Geral
O Dispatcher deve atuar como o núcleo de comunicação assíncrona/síncrona, garantindo a entrega isolada e segura de eventos.

### 2. Objetivos e Sucesso
- **Concorrência:** Operações de `Register`, `Dispatch` e `Remove` devem ser thread-safe.
- **Resiliência:** Implementar mecanismos de `recovery` para evitar que panics em handlers afetem o fluxo principal.
- **Performance:** Minimizar alocações e latência no roteamento.
- **Limpeza:** Garantir que `Remove` e `Clear` permitam a coleta de lixo (GC) das referências.

### 3. Funcionalidades Core
- Registro dinâmico de múltiplos handlers por evento.
- Despacho com suporte a `context.Context` (timeout/cancelamento).
- Verificação (`Has`) e remoção unitária (`Remove`) de handlers.
- Limpeza total do estado (`Clear`).

### 4. Restrições Técnicas de Alto Nível
- Uso de primitivas de sincronização eficientes (ex: `sync.RWMutex`).
- Isolamento: Handlers lentos ou com erro não devem travar outros handlers.
- Observabilidade: Facilitar tracing/logs sem impacto crítico em performance.

### 5. Fora de Escopo
- Persistência externa (out-of-process brokers como Kafka/RabbitMQ).
- Garantias "Exactly-once" distribuídas fora da memória do processo.

---

**Instrução para o Agente:**
Gere o PRD seguindo rigorosamente o template em `.claude/templates/prd-template.md`. Foque em detalhar requisitos não funcionais de performance e segurança de memória.
