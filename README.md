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
- [Módulo: Observabilidade (`pkg/o11y`)](#-módulo-observabilidade-pkgo11y)
  - [Arquitetura dos Pacotes](#arquitetura-dos-pacotes)
  - [Início Rápido — Fachada Unificada](#início-rápido--fachada-unificada)
  - [Configuração (`o11y.Config`)](#configuração-o11yconfig)
  - [Opções Disponíveis](#opções-disponíveis)
  - [Transporte OTLP gRPC](#transporte-otlp-grpc)
  - [Transporte OTLP HTTP](#transporte-otlp-http)
  - [Rastreamento (Tracing)](#rastreamento-tracing)
  - [Métricas (Metrics)](#métricas-metrics)
  - [Logs Estruturados (Logging)](#logs-estruturados-logging)
  - [Uso por Sinal (Signal Provider)](#uso-por-sinal-signal-provider)
  - [Handler de Log Customizado](#handler-de-log-customizado)
  - [Propagação de Contexto](#propagação-de-contexto)
  - [Observabilidade Zero — `noop`](#observabilidade-zero--noop)
  - [Testes com `oteltest`](#testes-com-oteltest)
  - [Exemplo Completo: HTTP Server](#exemplo-completo-http-server)
- [Desenvolvimento](#️-desenvolvimento)
- [Licença](#-licença)

---

## 📦 Módulos

| Módulo | Pacote | Descrição |
| :--- | :--- | :--- |
| Database Manager | `pkg/database` | Pool de conexão configurável, multi-driver, shutdown gracioso |
| Unit of Work | `pkg/database/uow` | Transações coordenadas via `.Do()`, rollback automático |
| Migrations | `pkg/database/migrate` | Migrations Up/Down com `golang-migrate` e `embed.FS` |
| Observabilidade | `pkg/o11y` | Tracing, Metrics e Logging via OpenTelemetry |

---

## 📦 Instalação

```bash
go get devkit
```

---

## 🗄️ Módulo: Database (`pkg/database`)

Gerencia connection pool para Postgres, MySQL e SQL Server. Não expõe abstrações sobre `database/sql` — o consumidor recebe o `*sql.DB` nativo e usa diretamente.

### Registro de Driver

Cada driver é um sub-pacote opcional. Importe via side-effect no `main.go` o driver que seu projeto usa:

```go
import _ "devkit/pkg/database/postgres"   // PostgreSQL
import _ "devkit/pkg/database/mysql"      // MySQL / MariaDB
import _ "devkit/pkg/database/sqlserver"  // SQL Server
```

O registro acontece no `init()` de cada sub-pacote. A chamada a `database.New` valida se o driver foi registrado e retorna `ErrUnsupportedDriver` caso contrário.

---

### Início Rápido — Database Manager

```go
import (
    "context"
    "log"

    "devkit/pkg/database"
    _ "devkit/pkg/database/postgres"
)

func main() {
    ctx := context.Background()

    mgr, err := database.New(ctx, database.Config{
        Driver: "postgres",
        DSN:    "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
    })
    if err != nil {
        log.Fatalf("database.New: %v", err)
    }
    defer func() { _ = mgr.Close(ctx) }()

    // *sql.DB nativo disponível para queries diretas
    rows, err := mgr.DB().QueryContext(ctx, "SELECT id, name FROM users WHERE active = $1", true)
    if err != nil {
        log.Fatalf("query: %v", err)
    }
    defer func() { _ = rows.Close() }()

    for rows.Next() {
        var id int
        var name string
        if err := rows.Scan(&id, &name); err != nil {
            log.Fatalf("scan: %v", err)
        }
        log.Printf("user %d: %s", id, name)
    }
    if err := rows.Err(); err != nil {
        log.Fatalf("rows: %v", err)
    }
}
```

#### MySQL

```go
import (
    "devkit/pkg/database"
    _ "devkit/pkg/database/mysql"
)

mgr, err := database.New(ctx, database.Config{
    Driver: "mysql",
    DSN:    "user:pass@tcp(localhost:3306)/mydb?parseTime=true",
})
```

#### SQL Server

```go
import (
    "devkit/pkg/database"
    _ "devkit/pkg/database/sqlserver"
)

mgr, err := database.New(ctx, database.Config{
    Driver: "sqlserver",
    DSN:    "sqlserver://user:pass@localhost:1433?database=mydb",
})
```

---

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
    database.WithConnMaxIdleTime(2*time.Minute),
)
```

Os mesmos parâmetros podem ser informados diretamente no `Config` quando preferir evitar Options:

```go
mgr, err := database.New(ctx, database.Config{
    Driver:          "postgres",
    DSN:             dsn,
    MaxOpenConns:    50,
    MaxIdleConns:    10,
    ConnMaxLifetime: 10 * time.Minute,
    ConnMaxIdleTime: 2 * time.Minute,
})
```

#### Shutdown gracioso com timeout

`Close` é idempotente e respeita cancelamento de contexto:

```go
shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := mgr.Close(shutCtx); err != nil {
    log.Printf("close: %v", err)
}
```

---

### Erros Sentinela — `pkg/database`

| Erro | Quando ocorre |
| :--- | :--- |
| `ErrDriverRequired` | `Config.Driver` está vazio |
| `ErrDSNRequired` | `Config.DSN` está vazio |
| `ErrUnsupportedDriver` | Driver não registrado via import side-effect |
| `ErrInvalidPoolConfig` | `MaxIdleConns > MaxOpenConns` ou valores negativos |

```go
mgr, err := database.New(ctx, database.Config{Driver: "postgres"})
if errors.Is(err, database.ErrDSNRequired) {
    log.Fatal("DSN obrigatório")
}

mgr, err = database.New(ctx, database.Config{Driver: "oracle", DSN: dsn})
if errors.Is(err, database.ErrUnsupportedDriver) {
    log.Fatal("driver não registrado — adicione o import side-effect")
}
```

---

### Unit of Work (`pkg/database/uow`)

Coordena múltiplos repositórios em uma única transação. Commit automático em sucesso, rollback automático em erro ou panic.

#### Interface `Querier`

`uow.Querier` é a abstração que permite que repositórios funcionem tanto dentro quanto fora de uma transação sem alterar código:

```go
// Querier é satisfeito por *sql.DB e *sql.Tx.
type Querier interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
    PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
```

Defina seus repositórios recebendo `uow.Querier`:

```go
type UserRepository struct {
    q uow.Querier
}

func NewUserRepository(q uow.Querier) *UserRepository {
    return &UserRepository{q: q}
}

func (r *UserRepository) Save(ctx context.Context, name string) error {
    _, err := r.q.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", name)
    return err
}

func (r *UserRepository) FindByID(ctx context.Context, id int) (string, error) {
    var name string
    err := r.q.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", id).Scan(&name)
    return name, err
}
```

#### Uso básico

```go
import (
    "database/sql"
    "devkit/pkg/database/uow"
)

u, err := uow.New(mgr.DB())
if err != nil {
    log.Fatalf("uow.New: %v", err)
}

// Registra factory: chamada a cada transação com o *sql.Tx ativo
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
if err != nil {
    log.Printf("transação falhou: %v", err)
}
```

#### Múltiplos repositórios em uma única transação

```go
type OrderRepository struct{ q uow.Querier }
type StockRepository struct{ q uow.Querier }

func (r *OrderRepository) Create(ctx context.Context, userID int, total float64) (int64, error) {
    var id int64
    err := r.q.QueryRowContext(ctx,
        "INSERT INTO orders (user_id, total) VALUES ($1, $2) RETURNING id",
        userID, total,
    ).Scan(&id)
    return id, err
}

func (r *StockRepository) Deduct(ctx context.Context, productID, qty int) error {
    _, err := r.q.ExecContext(ctx,
        "UPDATE stock SET quantity = quantity - $1 WHERE product_id = $2 AND quantity >= $1",
        qty, productID,
    )
    return err
}

// Registro
u.Register("orders", func(tx *sql.Tx) any { return &OrderRepository{q: tx} })
u.Register("stock",  func(tx *sql.Tx) any { return &StockRepository{q: tx} })

// Execução — ambos os repositórios operam no mesmo *sql.Tx
err = u.Do(ctx, func(ctx context.Context) error {
    orders, _ := uow.GetRepository[*OrderRepository](ctx, u, "orders")
    stock,  _ := uow.GetRepository[*StockRepository](ctx, u, "stock")

    orderID, err := orders.Create(ctx, 42, 199.90)
    if err != nil {
        return err // rollback automático
    }
    _ = orderID

    return stock.Deduct(ctx, 7, 1)
    // se Deduct falhar → rollback em ambas as operações
})
```

#### Acesso direto ao `*sql.Tx` via contexto

Quando não há repositório registrado, use `uow.TxFromContext` para obter a transação ativa:

```go
err = u.Do(ctx, func(ctx context.Context) error {
    tx := uow.TxFromContext(ctx) // retorna nil se chamado fora de Do
    if tx == nil {
        return fmt.Errorf("sem transação ativa")
    }
    _, err := tx.ExecContext(ctx, "INSERT INTO audit_log (msg) VALUES ($1)", "operação X")
    return err
})
```

#### Nível de isolamento customizado

```go
u, err := uow.New(mgr.DB(),
    uow.WithTxOptions(&sql.TxOptions{
        Isolation: sql.LevelSerializable,
        ReadOnly:  false,
    }),
)
```

**Opções de `uow.New`:**

| Opção | Default | Descrição |
| :--- | :--- | :--- |
| `WithTxOptions(opts)` | nil (driver default) | Nível de isolamento e modo de leitura/escrita |

#### Erros sentinela — `pkg/database/uow`

| Erro | Quando ocorre |
| :--- | :--- |
| `ErrDBRequired` | `db` nil passado a `New` |
| `ErrRepositoryNotFound` | Nome não registrado em `GetRepository` |
| `ErrNoActiveTransaction` | `GetRepository` ou `TxFromContext` chamados fora de `Do` |

```go
err = u.Do(ctx, func(ctx context.Context) error {
    _, err := uow.GetRepository[*UserRepository](ctx, u, "nonexistent")
    if errors.Is(err, uow.ErrRepositoryNotFound) {
        return fmt.Errorf("repositório não registrado: %w", err)
    }
    return err
})
```

---

### Migrations (`pkg/database/migrate`)

Executa migrations Up/Down via `golang-migrate`. Recebe um `*sql.DB` existente e um `fs.FS` com os arquivos SQL — compatível com `embed.FS` para embutir as migrations no binário.

#### Estrutura de arquivos

```
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_add_email_to_users.up.sql
└── 000002_add_email_to_users.down.sql
```

Exemplo de conteúdo:

```sql
-- 000001_create_users.up.sql
CREATE TABLE users (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

-- 000001_create_users.down.sql
DROP TABLE users;

-- 000002_add_email_to_users.up.sql
ALTER TABLE users ADD COLUMN email TEXT;

-- 000002_add_email_to_users.down.sql
ALTER TABLE users DROP COLUMN email;
```

#### Uso com `embed.FS`

```go
import (
    "embed"
    "io/fs"
    "log"

    "devkit/pkg/database/migrate"
)

//go:embed migrations
var migrationsFS embed.FS

func runMigrations(db *sql.DB) {
    sub, err := fs.Sub(migrationsFS, "migrations")
    if err != nil {
        log.Fatalf("fs.Sub: %v", err)
    }

    m, err := migrate.New(db, sub, migrate.Config{DatabaseDriver: "postgres"})
    if err != nil {
        log.Fatalf("migrate.New: %v", err)
    }
    defer func() { _ = m.Close() }()

    if err := m.Up(ctx); err != nil {
        log.Fatalf("migrate up: %v", err)
    }
    log.Println("migrations aplicadas com sucesso")
}
```

#### Rollback completo

```go
if err := m.Down(ctx); err != nil {
    log.Fatalf("migrate down: %v", err)
}
log.Println("todas as migrations revertidas")
```

#### Tabela de controle customizada

Por padrão o `golang-migrate` usa `schema_migrations`. Para customizar:

```go
m, err := migrate.New(db, sub, migrate.Config{DatabaseDriver: "postgres"},
    migrate.WithMigrationsTable("myapp_migrations"),
)
```

#### Tratamento de estado dirty

Se o processo morrer no meio de uma migration, a tabela fica em estado "dirty". O erro `ErrDirtyDatabase` carrega a versão problemática:

```go
if err := m.Up(ctx); err != nil {
    if errors.Is(err, migrate.ErrDirtyDatabase) {
        log.Printf("migration em estado dirty: %v — intervenção manual necessária", err)
        // Corrija o estado no banco e execute Force() se necessário
        return
    }
    log.Fatalf("migrate up: %v", err)
}
```

**Opções de `migrate.New`:**

| Opção | Default | Descrição |
| :--- | :--- | :--- |
| `WithMigrationsTable(name)` | `"schema_migrations"` | Nome da tabela de controle de versão |

**Erros sentinela — `pkg/database/migrate`:**

| Erro | Quando ocorre |
| :--- | :--- |
| `ErrDatabaseRequired` | `db` nil ou `Config.DatabaseDriver` vazio |
| `ErrSourceRequired` | `fsys` nil |
| `ErrDirtyDatabase` | Tabela de migrations em estado dirty (inclui número da versão) |

---

### Stack Completa — Database + Migrate + UoW

Composição típica na inicialização de uma aplicação:

```go
package main

import (
    "context"
    "database/sql"
    "embed"
    "io/fs"
    "log"
    "os"
    "os/signal"
    "syscall"

    "devkit/pkg/database"
    _ "devkit/pkg/database/postgres"
    "devkit/pkg/database/migrate"
    "devkit/pkg/database/uow"
)

//go:embed migrations
var migrationsFS embed.FS

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // 1. Connection pool
    mgr, err := database.New(ctx, database.Config{
        Driver: "postgres",
        DSN:    os.Getenv("DATABASE_URL"),
    },
        database.WithMaxOpenConns(50),
        database.WithMaxIdleConns(10),
    )
    if err != nil {
        log.Fatalf("database.New: %v", err)
    }
    defer func() { _ = mgr.Close(ctx) }()

    // 2. Migrations
    sub, _ := fs.Sub(migrationsFS, "migrations")
    migrator, err := migrate.New(mgr.DB(), sub, migrate.Config{DatabaseDriver: "postgres"})
    if err != nil {
        log.Fatalf("migrate.New: %v", err)
    }
    defer func() { _ = migrator.Close() }()

    if err := migrator.Up(ctx); err != nil {
        log.Fatalf("migrate up: %v", err)
    }

    // 3. Unit of Work
    unit, err := uow.New(mgr.DB())
    if err != nil {
        log.Fatalf("uow.New: %v", err)
    }
    unit.Register("users", func(tx *sql.Tx) any {
        return NewUserRepository(tx)
    })

    // 4. Execução transacional
    if err := unit.Do(ctx, func(ctx context.Context) error {
        repo, err := uow.GetRepository[*UserRepository](ctx, unit, "users")
        if err != nil {
            return err
        }
        return repo.Save(ctx, "Alice")
    }); err != nil {
        log.Printf("transação falhou: %v", err)
    }
}
```

---

### Testes de Integração (Database)

Os testes de integração usam [testcontainers-go](https://golang.testcontainers.org/) e requerem Docker. São separados dos testes unitários via build tag:

```bash
# Unitários (padrão, sem Docker)
make test

# Integração (requer Docker — Postgres e MySQL)
make test-integration

# SQL Server requer flag extra (imagem grande)
RUN_SQLSERVER_INTEGRATION=1 make test-integration
```

---

## 📡 Módulo: Observabilidade (`pkg/o11y`)

Fachada unificada e opinativa sobre o SDK oficial do OpenTelemetry. Inicializa Tracing, Metrics e Logging em uma única chamada com consistência de atributos de recurso entre todos os sinais, sem registrar estado global.

### Arquitetura dos Pacotes

```
pkg/o11y/
├── o11y             — Fachada principal: inicializa os três sinais juntos
├── tracing          — Signal provider de rastreamento isolado
├── metrics          — Signal provider de métricas isolado
├── logging          — Signal provider de logs estruturados isolado
├── otlpgrpc         — Adaptador OTLP gRPC unificado (três sinais)
├── otlphttp         — Adaptador OTLP HTTP unificado (três sinais)
├── tracing/otlpgrpc — Adaptador OTLP gRPC apenas para tracing
├── tracing/otlphttp — Adaptador OTLP HTTP apenas para tracing
├── metrics/otlpgrpc — Adaptador OTLP gRPC apenas para metrics
├── metrics/otlphttp — Adaptador OTLP HTTP apenas para metrics
├── logging/otlpgrpc — Adaptador OTLP gRPC apenas para logging
├── logging/otlphttp — Adaptador OTLP HTTP apenas para logging
├── noop             — Implementações zero-custo para ambientes sem exporters
└── oteltest         — FakeTracer, FakeMeter e FakeLogger para testes unitários
```

Adapters de transporte (`otlpgrpc`, `otlphttp` e variantes por sinal) ficam em pacotes separados para evitar que dependências transitivas pesadas (gRPC, protobuf) sejam carregadas por quem não precisa delas.

### Início Rápido — Fachada Unificada

```go
import (
    "devkit/pkg/o11y"
    "devkit/pkg/o11y/otlpgrpc"
)

func main() {
    ctx := context.Background()

    sdk, err := o11y.New(ctx, o11y.Config{
        ServiceName:    "order-api",
        ServiceVersion: "1.2.0",
        Environment:    "production",
    },
        otlpgrpc.WithTrace("otel-collector:4317"),
        otlpgrpc.WithMetric("otel-collector:4317"),
        otlpgrpc.WithLog("otel-collector:4317"),
        o11y.WithW3CPropagators(),
    )
    if err != nil {
        log.Fatalf("falha ao configurar o11y: %v", err)
    }
    defer func() { _ = sdk.Shutdown(ctx) }()

    tracer  := sdk.TracerProvider().Tracer("order-api")
    meter   := sdk.MeterProvider().Meter("order-api")
    logger  := sdk.Logger()

    logger.Info("aplicação iniciada com observabilidade completa!")
    _ = tracer
    _ = meter
}
```

### Configuração (`o11y.Config`)

| Campo | Obrigatório | Descrição |
| :--- | :--- | :--- |
| `ServiceName` | Sim | Identificador do serviço (`service.name` no resource) |
| `ServiceVersion` | Não | Versão da aplicação — ex: `"1.2.0"` ou tag git |
| `Environment` | Não | Ambiente de execução — ex: `"production"`, `"staging"` |
| `ResourceAttributes` | Não | Atributos `attribute.KeyValue` adicionais ao resource |

Os campos `TraceExporter`, `MetricExporter`, `LogExporter`, `TraceSampler`, `MetricInterval` e `Handler` também podem ser definidos diretamente no `Config` quando não for conveniente usar Options.

### Opções Disponíveis

| Opção | Descrição |
| :--- | :--- |
| `otlpgrpc.WithTrace(endpoint)` | Exporter de Trace via OTLP gRPC |
| `otlpgrpc.WithMetric(endpoint)` | Exporter de Metrics via OTLP gRPC |
| `otlpgrpc.WithLog(endpoint)` | Exporter de Logs via OTLP gRPC |
| `otlphttp.WithTrace(endpoint)` | Exporter de Trace via OTLP HTTP |
| `otlphttp.WithMetric(endpoint)` | Exporter de Metrics via OTLP HTTP |
| `otlphttp.WithLog(endpoint)` | Exporter de Logs via OTLP HTTP |
| `o11y.WithSampler(sampler)` | Estratégia de amostragem customizada |
| `o11y.WithMetricInterval(d)` | Intervalo de coleta de métricas (default: 60s) |
| `o11y.WithHandler(handler)` | Compõe `slog.Handler` customizado com a bridge OTel |
| `o11y.WithPropagator(prop)` | Substitui o propagador interno da fachada |
| `o11y.WithW3CPropagators()` | Registra W3C TraceContext + Baggage globalmente |

O `endpoint` é opcional em todos os adaptadores. Se omitido, o SDK lê `OTEL_EXPORTER_OTLP_ENDPOINT`.

### Transporte OTLP gRPC

```go
import (
    "devkit/pkg/o11y"
    "devkit/pkg/o11y/otlpgrpc"
)

sdk, err := o11y.New(ctx, o11y.Config{
    ServiceName:    "payment-service",
    ServiceVersion: "2.0.0",
    Environment:    "production",
},
    otlpgrpc.WithTrace("otel-collector:4317"),
    otlpgrpc.WithMetric("otel-collector:4317"),
    otlpgrpc.WithLog("otel-collector:4317"),
)
```

### Transporte OTLP HTTP

```go
import (
    "devkit/pkg/o11y"
    "devkit/pkg/o11y/otlphttp"
)

sdk, err := o11y.New(ctx, o11y.Config{
    ServiceName: "notification-service",
},
    otlphttp.WithTrace("http://otel-collector:4318"),
    otlphttp.WithMetric("http://otel-collector:4318"),
    otlphttp.WithLog("http://otel-collector:4318"),
)
```

### Rastreamento (Tracing)

```go
tracer := sdk.TracerProvider().Tracer("orders")

func processOrder(ctx context.Context, tracer trace.Tracer, orderID string) error {
    ctx, span := tracer.Start(ctx, "process-order",
        trace.WithAttributes(attribute.String("order.id", orderID)),
    )
    defer span.End()

    if err := validateStock(ctx, orderID); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "stock validation failed")
        return err
    }

    span.SetAttributes(attribute.Bool("order.stock_ok", true))
    return nil
}
```

#### Amostragem customizada

```go
import sdktrace "go.opentelemetry.io/otel/sdk/trace"

sdk, err := o11y.New(ctx, o11y.Config{
    ServiceName: "high-traffic-api",
},
    otlpgrpc.WithTrace("otel-collector:4317"),
    o11y.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // 10% dos traces
)
```

#### Sampler pai-baseado (padrão de produção)

```go
o11y.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.05)))
```

### Métricas (Metrics)

```go
meter := sdk.MeterProvider().Meter("payments")

// Contador
requestsTotal, _ := meter.Int64Counter("http_requests_total",
    metric.WithDescription("Total de requisições HTTP recebidas"),
    metric.WithUnit("{request}"),
)
requestsTotal.Add(ctx, 1,
    metric.WithAttributes(
        attribute.String("method", "POST"),
        attribute.String("path", "/orders"),
        attribute.Int("status_code", 200),
    ),
)

// Histograma de latência
latency, _ := meter.Float64Histogram("http_request_duration_seconds",
    metric.WithDescription("Duração das requisições HTTP em segundos"),
    metric.WithUnit("s"),
)
latency.Record(ctx, 0.042,
    metric.WithAttributes(attribute.String("handler", "create-order")),
)

// Gauge
activeConnections, _ := meter.Int64UpDownCounter("db_active_connections",
    metric.WithDescription("Conexões ativas no pool"),
)
activeConnections.Add(ctx, 1)
```

#### Intervalo de coleta customizado

```go
sdk, err := o11y.New(ctx, o11y.Config{
    ServiceName: "analytics-service",
},
    otlpgrpc.WithMetric("otel-collector:4317"),
    o11y.WithMetricInterval(15*time.Second),
)
```

### Logs Estruturados (Logging)

O logger retornado é um `*slog.Logger` padrão. Quando um exporter OTel de logs está configurado, cada registro é enviado automaticamente ao backend via bridge OTLP. Quando o logger é usado com um `context.Context` que carrega um span ativo, `trace_id` e `span_id` são injetados automaticamente no registro.

```go
logger := sdk.Logger()

// Log simples com atributos estruturados
logger.Info("usuário autenticado",
    "user_id", 42,
    "ip", "192.168.1.1",
)

// Log com context — injeta trace_id e span_id automaticamente
logger.InfoContext(ctx, "pedido criado",
    "order_id", "ord-123",
    "total_cents", 9900,
)

// Log de erro com causa
logger.ErrorContext(ctx, "falha ao processar pagamento",
    "payment_id", "pay-456",
    "error", err,
)

// Log condicional por nível
if logger.Enabled(ctx, slog.LevelDebug) {
    logger.DebugContext(ctx, "payload recebido", "body", rawBody)
}
```

### Uso por Sinal (Signal Provider)

Para cenários que precisam apenas de um sinal, cada pacote de sinal pode ser usado de forma independente.

#### Somente Tracing

```go
import (
    "devkit/pkg/o11y/tracing"
    tracinggrpc "devkit/pkg/o11y/tracing/otlpgrpc"
)

p, err := tracing.New(ctx, tracing.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
},
    tracinggrpc.WithOTLPGRPC("otel-collector:4317"),
    tracing.WithSampler(sdktrace.AlwaysSample()),
)
if err != nil { ... }
defer func() { _ = p.Shutdown(ctx) }()

tracer := p.TracerProvider().Tracer("my-service")
```

#### Somente Metrics

```go
import (
    "devkit/pkg/o11y/metrics"
    metricsgrpc "devkit/pkg/o11y/metrics/otlpgrpc"
)

p, err := metrics.New(ctx, metrics.Config{
    ServiceName: "my-service",
},
    metricsgrpc.WithOTLPGRPC("otel-collector:4317"),
    metrics.WithInterval(30*time.Second),
)
if err != nil { ... }
defer func() { _ = p.Shutdown(ctx) }()

meter := p.MeterProvider().Meter("my-service")
```

#### Somente Logging

```go
import (
    "devkit/pkg/o11y/logging"
    logginggrpc "devkit/pkg/o11y/logging/otlpgrpc"
)

p, err := logging.New(ctx, logging.Config{
    ServiceName: "my-service",
},
    logginggrpc.WithOTLPGRPC("otel-collector:4317"),
)
if err != nil { ... }
defer func() { _ = p.Shutdown(ctx) }()

logger := p.Logger()
logger.Info("iniciado")
```

#### Via OTLP HTTP por sinal

```go
import (
    tracinghttp  "devkit/pkg/o11y/tracing/otlphttp"
    metricshttp  "devkit/pkg/o11y/metrics/otlphttp"
    logginghttp  "devkit/pkg/o11y/logging/otlphttp"
)

// Cada sinal pode apontar para endpoints distintos
p, err := tracing.New(ctx, tracing.Config{ServiceName: "my-service"},
    tracinghttp.WithOTLPHTTP("http://tempo:4318"),
)
```

### Handler de Log Customizado

O pacote suporta composição de handlers: o exporter OTel e um handler customizado recebem cada registro simultaneamente.

```go
import (
    "log/slog"
    "os"
    "devkit/pkg/o11y"
    "devkit/pkg/o11y/otlpgrpc"
)

// Escreve em JSON no stdout E envia para o coletor OTLP
jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})

sdk, err := o11y.New(ctx, o11y.Config{
    ServiceName: "my-service",
},
    otlpgrpc.WithLog("otel-collector:4317"),
    o11y.WithHandler(jsonHandler),
)
```

### Propagação de Contexto

A fachada usa W3C TraceContext + Baggage internamente. Para propagar contexto via HTTP, registre globalmente:

```go
sdk, err := o11y.New(ctx, cfg,
    otlpgrpc.WithTrace("otel-collector:4317"),
    o11y.WithW3CPropagators(), // registra otel.SetTextMapPropagator globalmente
)

// Injeção em requisição de saída
req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

// Extração em requisição de entrada
ctx = otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
```

Quando a propagação global não for desejada, use o propagador da própria fachada:

```go
prop := sdk.Propagator()
prop.Inject(ctx, propagation.HeaderCarrier(req.Header))
ctx  = prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
```

### Resource Attributes Customizados

```go
import "go.opentelemetry.io/otel/attribute"

sdk, err := o11y.New(ctx, o11y.Config{
    ServiceName:    "checkout-api",
    ServiceVersion: "3.1.0",
    Environment:    "production",
    ResourceAttributes: []attribute.KeyValue{
        attribute.String("team", "platform"),
        attribute.String("region", "us-east-1"),
        attribute.String("cloud.provider", "aws"),
    },
})
```

### Observabilidade Zero — `noop`

Use `noop` em testes ou em contextos onde observabilidade não deve ter overhead:

```go
import "devkit/pkg/o11y/noop"

tracerProvider := noop.NewTracerProvider()
meterProvider  := noop.NewMeterProvider()
logger          := noop.NewLogger()

// Todas as operações são no-op — nenhuma alocação relevante
tracer := tracerProvider.Tracer("my-service")
_, span := tracer.Start(ctx, "op")
span.End()
```

### Testes com `oteltest`

O pacote `oteltest` fornece doubles em memória para os três sinais, permitindo asserções precisas sem infraestrutura externa.

#### FakeTracer

```go
import "devkit/pkg/o11y/oteltest"

func TestProcessOrder(t *testing.T) {
    fake := oteltest.NewFakeTracer()

    tracer := fake.Tracer("orders")
    ctx, span := tracer.Start(context.Background(), "process-order")
    span.SetAttributes(attribute.String("order.id", "ord-123"))
    span.End()

    _ = ctx // ctx propagado para sub-operações

    spans := fake.Spans()
    if len(spans) != 1 {
        t.Fatalf("esperava 1 span, obteve %d", len(spans))
    }
    if spans[0].Name != "process-order" {
        t.Errorf("nome inesperado: %s", spans[0].Name)
    }

    // Limpar entre sub-testes
    fake.Reset()
}
```

#### FakeMeter

```go
func TestRecordPayment(t *testing.T) {
    fake := oteltest.NewFakeMeter()

    meter := fake.MeterProvider().Meter("payments")
    counter, _ := meter.Int64Counter("payments_total")
    counter.Add(context.Background(), 3)

    rm, err := fake.Collect(context.Background())
    if err != nil {
        t.Fatal(err)
    }

    // rm.ScopeMetrics contém as métricas coletadas
    if len(rm.ScopeMetrics) == 0 {
        t.Fatal("nenhuma métrica coletada")
    }
}
```

#### FakeLogger

```go
func TestAuditLog(t *testing.T) {
    fake := oteltest.NewFakeLogger()

    logger := fake.Logger()
    logger.Info("usuário criado", "user_id", 99)
    logger.Warn("tentativa suspeita", "ip", "10.0.0.1")

    records := fake.Records()
    if len(records) != 2 {
        t.Fatalf("esperava 2 registros, obteve %d", len(records))
    }
    if records[0].Message != "usuário criado" {
        t.Errorf("mensagem inesperada: %s", records[0].Message)
    }

    fake.Reset()
}
```

#### Usando TracerProvider diretamente

```go
func TestHTTPHandler(t *testing.T) {
    fake := oteltest.NewFakeTracer()

    // Injeta o provider no handler sob teste
    handler := NewOrderHandler(fake.TracerProvider())

    req := httptest.NewRequest("POST", "/orders", body)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    spans := fake.Spans()
    // Valida que o handler criou o span esperado
    require.Len(t, spans, 1)
    assert.Equal(t, "create-order", spans[0].Name)
}
```

### Exemplo Completo: HTTP Server

Exemplo de integração real com `net/http`, combinando os três sinais:

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"

    "devkit/pkg/o11y"
    "devkit/pkg/o11y/otlpgrpc"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    sdk, err := o11y.New(ctx, o11y.Config{
        ServiceName:    "order-api",
        ServiceVersion: os.Getenv("APP_VERSION"),
        Environment:    os.Getenv("APP_ENV"),
    },
        otlpgrpc.WithTrace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
        otlpgrpc.WithMetric(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
        otlpgrpc.WithLog(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
        o11y.WithW3CPropagators(),
    )
    if err != nil {
        log.Fatalf("o11y: %v", err)
    }
    defer func() {
        shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        _ = sdk.Shutdown(shutCtx)
    }()

    tracer := sdk.TracerProvider().Tracer("order-api")
    meter  := sdk.MeterProvider().Meter("order-api")
    logger := sdk.Logger()

    reqs, _ := meter.Int64Counter("http_requests_total")
    lat, _  := meter.Float64Histogram("http_request_duration_seconds", metric.WithUnit("s"))

    mux := http.NewServeMux()
    mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Extrai contexto de propagação do cabeçalho
        ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

        ctx, span := tracer.Start(ctx, "create-order",
            trace.WithAttributes(attribute.String("http.method", r.Method)),
        )
        defer span.End()

        logger.InfoContext(ctx, "criando pedido")

        // lógica de negócio...
        w.WriteHeader(http.StatusCreated)

        elapsed := time.Since(start).Seconds()
        attrs   := metric.WithAttributes(attribute.Int("status_code", 201))
        reqs.Add(ctx, 1, attrs)
        lat.Record(ctx, elapsed, attrs)

        logger.InfoContext(ctx, "pedido criado", slog.Float64("duration_s", elapsed))
    })

    srv := &http.Server{Addr: ":8080", Handler: mux}

    go func() {
        logger.Info("servidor iniciado", "addr", ":8080")
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("falha no servidor", "error", err)
        }
    }()

    <-ctx.Done()
    logger.Info("encerrando servidor")

    shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = srv.Shutdown(shutCtx)
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
