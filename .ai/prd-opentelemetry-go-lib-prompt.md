# Prompt para iniciar o PRD

Crie um PRD completo para uma biblioteca Go cujo objetivo seja encapsular a instrumentacao do OpenTelemetry para Go, tomando como referencia principal:

- https://opentelemetry.io/docs/languages/go/instrumentation/
- https://refactoring.guru/design-patterns

O PRD deve ser pragmatico, tecnico e orientado a implementacao. Evite texto generico. Se alguma decisao arquitetural nao estiver justificada, trate como opcao e registre trade-offs em vez de inventar complexidade.

## Objetivo do produto

Descrever uma library Go que ofereca uma interface unificada para observabilidade, concentrando:

- `logger`
- `tracing`
- `metrics`

A biblioteca deve permitir:

- importar tudo junto por uma interface principal
- importar modulos separadamente quando o consumidor quiser apenas `logger`, apenas `tracing` ou apenas `metrics`
- uso de `slog` quando fizer sentido
- disponibilizar implementacoes mockaveis/importaveis para testes
- disponibilizar implementacoes `noop` para ambientes sem observabilidade ativa

## Requisitos obrigatorios

O PRD precisa cobrir explicitamente:

1. Problema a resolver
2. Objetivos e nao objetivos
3. Personas/consumidores da biblioteca
4. Casos de uso principais
5. Requisitos funcionais
6. Requisitos nao funcionais
7. Arquitetura proposta
8. Estrategia de modularizacao dos pacotes
9. Design da API publica
10. Estrategia de mocks, fakes ou noops para testes
11. Integracao com `slog`
12. Decisoes de design patterns
13. Estrategia de configuracao e inicializacao
14. Compatibilidade e versionamento
15. Estrategia de testes
16. Riscos, trade-offs e decisoes em aberto
17. Roadmap de implementacao por fases
18. Criterios de aceite

## Diretrizes de arquitetura

Considere estas restricoes e preferencias:

- Nao criar codigo desnecessario nem abstractions over abstractions.
- A API deve ser simples para o caso comum e extensivel para casos avancados.
- O design deve separar claramente interface publica, implementacoes concretas e adapters.
- O consumidor deve poder escolher entre:
  - um facade/unified entrypoint
  - imports modulares por capability
- O PRD deve avaliar quando usar `Facade`, `Factory`, `Adapter`, `Strategy` ou outros patterns do refactoring.guru, mas somente se houver ganho real.
- Se algum pattern aumentar complexidade sem beneficio claro, o PRD deve rejeita-lo explicitamente.
- A biblioteca deve facilitar uso em producao, desenvolvimento local e testes.

## Expectativas para a API

O PRD deve propor uma API com exemplos conceituais para:

- bootstrap unico, por exemplo algo como `observability.New(...)`
- acesso agregado, por exemplo `obs.Logger()`, `obs.Tracer()`, `obs.Meter()`
- imports separados, por exemplo modulos independentes para logging, tracing e metrics
- providers `noop`
- providers de teste/mock importaveis
- integracao opcional com `slog.Handler` ou adaptadores equivalentes, se essa for a melhor abordagem

Nao escreva codigo final de implementacao. Use apenas pseudo-API, assinaturas sugestivas e exemplos curtos para explicar o desenho.

## Pontos tecnicos que o PRD deve responder

- Qual deve ser o package raiz?
- Como organizar subpackages sem poluir a API publica?
- Como expor a interface unificada sem acoplar excessivamente logger, tracing e metrics?
- Como permitir inicializacao parcial?
- Como representar dependencia de exporters/providers externos?
- Como estruturar `noop` e `mock` para serem ergonomicos e previsiveis?
- Como evitar lock-in prematuro em implementacoes concretas?
- Como alinhar a biblioteca com idioms de Go?
- Como garantir testabilidade sem proliferar interfaces artificiais?

## Formato de saida

Responda em Markdown com secoes claras e objetivas.

Inclua:

- uma proposta de estrutura de diretorios/pacotes
- uma proposta de API publica inicial
- exemplos de uso de alto nivel
- tabela de trade-offs arquiteturais
- lista de decisoes abertas
- checklist final de aceite

## Tom esperado

- tecnico
- direto
- sem marketing
- sem floreio
- com recomendacoes claras e justificadas

Se houver mais de uma boa alternativa, escolha uma recomendacao principal e apresente as demais apenas como comparacao objetiva.
