# DevKit

[![CI](https://github.com/jailtonjunior/devkit/actions/workflows/ci.yml/badge.svg)](https://github.com/jailtonjunior/devkit/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/devkit.svg)](https://pkg.go.dev/devkit)
[![Go Report Card](https://goreportcard.com/badge/github.com/jailtonjunior/devkit)](https://goreportcard.com/report/github.com/jailtonjunior/devkit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**DevKit** é uma biblioteca Go de padrão enterprise com módulos independentes para os problemas mais comuns em microserviços: observabilidade, conexão com banco de dados, transações coordenadas e migrations. Cada módulo pode ser importado individualmente — você paga apenas pelo que usa.

---

## 📑 Sumário

- [Módulos](#-módulos)
- [Instalação](#-instalação)
- [Módulo: Database (`pkg/database`)](#-módulo-database-pkgdatabase)
  - [Início Rápido — Database Manager](#início-rápido--database-manager)
  - [Opções de Pool](#opções-de-pool)
  - [Unit of Work (`pkg/database/uow`)](#unit-of-work-pkgdatabaseuow)
  - [Migrations (`pkg/database/migrate`)](#migrations-pkgdatabasemigrate)
  - [Testes de Integração](#testes-de-integração-database)
- [Módulo: Observabilidade (`o11y`)](#-módulo-observabilidade-o11y)
  - [Início Rápido — o11y](#início-rápido--o11y)
  - [Rastreamento (Tracing)](#rastreamento-tracing)
  - [Métricas (Metrics)](#métricas-metrics)
  - [Logs Estruturados (Logging)](#logs-estruturados-logging)
  - [Configuração e Opções](#configuração-e-opções)
  - [Testes e Mocking](#testes-e-mocking)
- [Desenvolvimento](#️-desenvolvimento)
- [Licença](#-licença)

---

## 📦 Módulos

| Módulo | Pacote | Descrição |
| :--- | :--- | :--- |
| Database Manager | `pkg/database` | Pool de conexão configurável, multi-driver, shutdown gracioso |
| Unit of Work | `pkg/database/uow` | Transações coordenadas via `.Do()`, rollback automático |
| Migrations | `pkg/database/migrate` | Migrations Up/Down com `golang-migrate` e `embed.FS` |
| Observabilidade | `o11y` | Tracing, Metrics e Logging via OpenTelemetry |

---

## 📦 Instalação

```bash
go get devkit
```

---

## 🗄️ Módulo: Database (`pkg/database`)

Gerencia connection pool para Postgres, MySQL e SQL Server. Não expõe abstrações sobre `database/sql` — o consumidor recebe o `*sql.DB` nativo e usa diretamente.

O consumidor registra o driver desejado via import side-effect no `main.go`:

```go
import _ "github.com/lib/pq"                      // postgres
import _ "github.com/go-sql-driver/mysql"          // mysql
import _ "github.com/microsoft/go-mssqldb"         // sqlserver
```

### Início Rápido — Database Manager

```go
import "devkit/pkg/database"

ctx := context.Background()

mgr, err := database.New(ctx, database.Config{
    Driver: "postgres",
    DSN:    "postgres://user:pass@localhost/mydb?sslmode=disable",
})
if err != nil {
    log.Fatalf("falha ao conectar: %v", err)
}
defer func() { _ = mgr.Close(ctx) }()

// Acesso direto ao *sql.DB nativo
rows, err := mgr.DB().QueryContext(ctx, "SELECT id, name FROM users")
```

### Opções de Pool

Os parâmetros de pool são opcionais. Os defaults cobrem a maioria dos casos:

| Opção | Default | Descrição |
| :--- | :--- | :--- |
| `WithMaxOpenConns(n)` | 25 | Máximo de conexões abertas |
| `WithMaxIdleConns(n)` | 5 | Máximo de conexões idle no pool |
| `WithConnMaxLifetime(d)` | 5m | Tempo máximo de reuso de uma conexão |
| `WithConnMaxIdleTime(d)` | 5m | Tempo máximo idle de uma conexão |

```go
mgr, err := database.New(ctx, database.Config{Driver: "postgres", DSN: dsn},
    database.WithMaxOpenConns(50),
    database.WithMaxIdleConns(10),
    database.WithConnMaxLifetime(10*time.Minute),
)
```

### Unit of Work (`pkg/database/uow`)

Coordena múltiplos repositórios em uma única transação. Commit automático em sucesso, rollback automático em erro ou panic.

```go
import "devkit/pkg/database/uow"

u, err := uow.New(db)
if err != nil { ... }

u.Register("users", func(tx *sql.Tx) any {
    return NewUserRepository(tx)
})

err = u.Do(ctx, func(ctx context.Context) error {
    repo, err := uow.GetRepository[*UserRepository](ctx, u, "users")
    if err != nil {
        return err
    }
    if err := repo.Save(ctx, "Alice"); err != nil {
        return err
    }
    return repo.Save(ctx, "Bob")
    // sucesso → commit automático
    // erro    → rollback automático
    // panic   → rollback + re-panic
})
```

Repositórios implementam a interface `uow.Querier`, que é satisfeita tanto por `*sql.DB` quanto por `*sql.Tx`. Isso permite usá-los fora de transação sem alteração de código:

```go
type UserRepository struct {
    q uow.Querier
}

func NewUserRepository(q uow.Querier) *UserRepository {
    return &UserRepository{q: q}
}
```

**Opções de `uow.New`:**

| Opção | Default | Descrição |
| :--- | :--- | :--- |
| `WithTxOptions(opts)` | nil | Nível de isolamento e modo de leitura/escrita das transações |

```go
u, err := uow.New(db,
    uow.WithTxOptions(&sql.TxOptions{
        Isolation: sql.LevelSerializable,
        ReadOnly:  false,
    }),
)
```

### Migrations (`pkg/database/migrate`)

Executa migrations Up/Down via `golang-migrate`. Recebe um `*sql.DB` existente e um `fs.FS` com os arquivos SQL — compatível com `embed.FS` para embutir as migrations no binário.

```go
import (
    "embed"
    "io/fs"
    "devkit/pkg/database/migrate"
)

//go:embed migrations
var migrationsFS embed.FS

sub, err := fs.Sub(migrationsFS, "migrations")
if err != nil { ... }

m, err := migrate.New(db, sub, migrate.Config{DatabaseDriver: "postgres"})
if err != nil { ... }
defer func() { _ = m.Close() }()

// Aplica todas as migrations pendentes
if err := m.Up(ctx); err != nil {
    log.Fatalf("migration falhou: %v", err)
}

// Reverte todas as migrations aplicadas
if err := m.Down(ctx); err != nil {
    log.Fatalf("rollback de migration falhou: %v", err)
}
```

Estrutura de arquivos de migration:

```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_add_email.up.sql
└── 000002_add_email.down.sql
```

**Opções de `migrate.New`:**

| Opção | Default | Descrição |
| :--- | :--- | :--- |
| `WithMigrationsTable(name)` | `"schema_migrations"` | Nome da tabela de controle de versão |

```go
m, err := migrate.New(db, sub, migrate.Config{DatabaseDriver: "postgres"},
    migrate.WithMigrationsTable("db_migrations"),
)
```

**Erros sentinela:**

| Erro | Quando ocorre |
| :--- | :--- |
| `ErrDatabaseRequired` | `db` nil ou `DatabaseDriver` vazio |
| `ErrSourceRequired` | `fsys` nil |
| `ErrDirtyDatabase` | Migration table em estado dirty — use `Force()` na instância subjacente |

### Testes de Integração (Database)

Os testes de integração usam [testcontainers-go](https://golang.testcontainers.org/) e requerem Docker. São separados dos testes unitários via build tag:

```bash
# Unitários (padrão, sem Docker)
make test

# Integração (requer Docker)
make test-integration
```

---

## 📡 Módulo: Observabilidade (`o11y`)

O **`o11y`** é uma fachada unificada e opinativa sobre o SDK oficial do OpenTelemetry. Inicializa Tracing, Metrics e Logging em uma única chamada, garantindo consistência de atributos de recurso entre todos os sinais.

### Início Rápido — o11y

```go
import (
    "devkit/o11y"
    "devkit/o11y/otlpgrpc"
)

sdk, err := o11y.New(ctx, o11y.Config{
    ServiceName:    "order-api",
    ServiceVersion: "1.0.0",
    Environment:    "production",
},
    otlpgrpc.WithTrace(),
    otlpgrpc.WithMetric(),
    otlpgrpc.WithLog(),
    o11y.WithW3CPropagators(),
)
if err != nil {
    log.Fatalf("falha ao configurar o11y: %v", err)
}
defer sdk.Shutdown(ctx)

logger := sdk.Logger()
logger.Info("aplicação iniciada com observabilidade completa!")
```

### Rastreamento (Tracing)

```go
func processOrder(ctx context.Context, sdk *o11y.Observability, orderID string) {
    tracer := sdk.TracerProvider().Tracer("orders")

    ctx, span := tracer.Start(ctx, "process-order")
    defer span.End()

    span.SetAttributes(attribute.String("order.id", orderID))

    // O logger injeta automaticamente trace_id e span_id
    sdk.Logger().InfoContext(ctx, "validando estoque")
}
```

### Métricas (Metrics)

```go
func recordPayment(ctx context.Context, sdk *o11y.Observability) {
    meter := sdk.MeterProvider().Meter("payments")

    counter, _ := meter.Int64Counter("payments_total",
        metric.WithDescription("total de pagamentos processados"),
    )
    counter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
}
```

### Logs Estruturados (Logging)

```go
logger := sdk.Logger()

logger.Info("usuário autenticado", "user_id", 42)

logger.ErrorContext(ctx, "falha na query",
    "db_host", "localhost",
    "error", err,
)
```

### Configuração e Opções

**`o11y.Config`:**

| Campo | Descrição |
| :--- | :--- |
| `ServiceName` | **(Obrigatório)** Identificador do serviço |
| `ServiceVersion` | Versão da aplicação (ex: tag git ou semver) |
| `Environment` | Ambiente (ex: `"prod"`, `"staging"`) |
| `ResourceAttributes` | Atributos `attribute.KeyValue` extras para o resource |

**Options disponíveis:**

| Opção | Descrição |
| :--- | :--- |
| `otlpgrpc.WithTrace(endpoint)` | Exporter de Trace via gRPC |
| `otlpgrpc.WithMetric(endpoint)` | Exporter de Metrics via gRPC |
| `otlpgrpc.WithLog(endpoint)` | Exporter de Logs via gRPC |
| `otlphttp.WithTrace(endpoint)` | Exporter de Trace via HTTP |
| `otlphttp.WithMetric(endpoint)` | Exporter de Metrics via HTTP |
| `otlphttp.WithLog(endpoint)` | Exporter de Logs via HTTP |
| `WithSampler(sampler)` | Estratégia de amostragem |
| `WithMetricInterval(d)` | Intervalo entre exportações de métricas (default: 60s) |
| `WithW3CPropagators()` | Ativa propagação W3C TraceContext + Baggage |

### Testes e Mocking

```go
import "devkit/o11y/oteltest"

func TestBusinessLogic(t *testing.T) {
    fake := oteltest.NewFakeTracer()

    tracer := fake.Tracer("test")
    _, span := tracer.Start(context.Background(), "op")
    span.End()

    spans := fake.Spans()
    if len(spans) != 1 {
        t.Errorf("esperava 1 span, obteve %d", len(spans))
    }
}
```

---

## 🛠️ Desenvolvimento

### Makefile

| Target | Descrição |
| :--- | :--- |
| `make tools` | Instala ferramentas ausentes (`golangci-lint`, `govulncheck`, `gosec`) |
| `make lint` | Análise estática com `golangci-lint` |
| `make test` | Testes unitários com race detector e `coverage.out` |
| `make test-integration` | Testes de integração com Docker (testcontainers) |
| `make security` | Varredura de vulnerabilidades (`govulncheck` + `gosec`) |
| `make ci` | `lint` + `test` + `security` em sequência |

```bash
make ci
```

### Conventional Commits

| Prefixo | Impacto | Exemplo |
| :--- | :--- | :--- |
| `fix:` | Patch (`v1.0.X`) | `fix: corrige rollback em panic no UoW` |
| `feat:` | Minor (`v1.X.0`) | `feat: adiciona WithMigrationsTable` |
| `feat!:` / `BREAKING CHANGE:` | Major (`vX.0.0`) | `feat!: remove API legada` |

---

## 🤝 Contribuição

Contribuições são bem-vindas. Abra uma issue ou envie um PR.

---

## 📄 Licença

Distribuído sob a licença MIT. Veja `LICENSE` para mais informações.
