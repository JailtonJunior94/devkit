# Prompt: Implementação do Job/Worker Manager em Go

## Objetivo
Criar um Gerenciador de Workers e Jobs (Cron) robusto no pacote `pkg/worker`, utilizando a biblioteca `github.com/robfig/cron/v3`. O sistema deve suportar processos contínuos (ex: Kafka consumers) e tarefas agendadas (cron), garantindo um ciclo de vida seguro (graceful shutdown) sem dependência do framework Uber FX.

## Requisitos Técnicos

### 1. Interfaces Core (`worker.go`)
Definir as seguintes interfaces para padronizar o comportamento:
- `Worker`: Para processos de longa duração.
  - `Start(ctx context.Context) error`
  - `Stop(ctx context.Context) error`
- `Job`: Para tarefas agendadas.
  - `Name() string`
  - `Schedule() string` (formato cron)
  - `Execute(ctx context.Context)`

### 2. Adaptador de Jobs (`adapter.go`)
Implementar um `jobAdapter` que permita transformar funções comuns em objetos que satisfaçam a interface `Job`.
- Deve incluir uma factory `NewJobAdapter(name, schedule string, fn func(ctx context.Context)) Job`.

### 3. Gerenciador (`manager.go`)
Implementar o `WorkerManager` com as seguintes características:
- **NÃO utilizar Uber FX**. O gerenciador deve ser agnóstico a frameworks de DI.
- **Campos**:
  - Lista de `Worker`.
  - Lista de `Job`.
  - Instância do `cron.Cron`.
  - `sync.WaitGroup` para rastrear goroutines ativas.
  - `context.CancelFunc` global para sinalizar parada.
- **Iniciação (`Start`)**:
  - Configurar `cron.New` com `cron.WithLocation` (America/Sao_Paulo) e `cron.WithChain(cron.SkipIfStillRunning)`.
  - Registrar todos os jobs no scheduler.
  - Iniciar cada `Worker` em sua própria goroutine.
  - Utilizar `slog` para logs de registro e início.
- **Parada (`Stop`)**:
  - Aceitar um `context.Context` com timeout (ex: 30s).
  - Parar o scheduler do cron.
  - Cancelar os contextos dos workers e jobs.
  - Aguardar a conclusão via `WaitGroup` ou estourar o timeout do contexto.
  - Reportar via logs quais workers/jobs não finalizaram a tempo.

### 4. Dependências
- `github.com/robfig/cron/v3`
- `log/slog` (padrão Go 1.21+)

## Padrões de Código
- **Idiomatismo Go**: Uso correto de contextos para cancelamento cooperativo.
- **Thread-Safety**: Garantir que a lista de workers e a parada sejam seguras contra race conditions.
- **Logs**: Incluir contexto nos logs (nome do job, erro, duração se possível).

## Exemplo de Assinatura do Construtor
```go
func NewWorkerManager(workers []Worker, jobs []Job) *WorkerManager
```

## Entregáveis
1. Arquivos no pacote `pkg/worker`.
2. Testes unitários validando o registro de jobs e o comportamento do shutdown.
3. Um exemplo de uso (pode ser em `example_test.go`) demonstrando como instanciar e parar o manager manualmente.
