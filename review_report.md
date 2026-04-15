# Relatório de Review

**Veredito**: REJECTED

Revisão executada em `2026-04-14 11:29:50 -03`.

## Achados Técnicos
### Critical
- `{ id: O11Y-001, severity: Critical, file: pkg/o11y/otel/config.go:67, line: 67, reproduction: "1. Execute `go run /tmp/o11y-invalid-protocol-XXXX.go` com `cfg.OTLPProtocol = otel.OTLPProtocol(\"invalid\")`; 2. Observe a saída `nil error`.", expected: "A configuração pública deve rejeitar protocolo OTLP inválido com erro seguro e acionável na fronteira pública.", actual: "O código normaliza qualquer valor desconhecido para `ProtocolGRPC` em `normalizeProtocol` (`pkg/o11y/otel/config.go:67-75`) e `validateConfig` não valida o campo (`pkg/o11y/otel/config.go:102-115`), então `otel.NewProvider` aceita configuração inválida sem erro." }`
  Ref de regra: `R-SEC-001` (configurações inválidas devem falhar cedo), `R-ARCH-001` (invariantes/configuração devem ser protegidas na criação).
- `{ id: O11Y-002, severity: Critical, file: pkg/o11y/logging/logging.go:51, line: 51, reproduction: "1. Execute `go run /tmp/o11y-global-logger-XXXX.go`; 2. O programa faz `slog.SetDefault(...)`, chama `logging.New(context.Background(), logging.Config{ServiceName: \"svc\"})` sem exporter/handler e `provider.Logger().Info(\"hello\")`; 3. Observe a linha sendo escrita pelo logger global alterado.", expected: "A biblioteca deve funcionar sem depender de logger global mutável; o provider retornado deveria ser isolado do `slog.Default()` ou exigir injeção explícita.", actual: "No caminho `cfg.LogExporter == nil`, `logging.New` retorna `slog.Default()` (`pkg/o11y/logging/logging.go:51-59`), acoplando o comportamento da API pública ao estado global mutável do processo." }`
  Ref de regra: `R-ARCH-001` (o núcleo da biblioteca não deve exigir logger global ou estado global mutável para funcionar).

### Major
- `{ id: O11Y-003, severity: Major, file: pkg/o11y/README.md:318, line: 318, reproduction: "1. Leia `pkg/o11y/README.md:318`, `pkg/o11y/README.md:336` e `pkg/o11y/README.md:410`; 2. Compare com `pkg/o11y/noop/noop.go:127-132`, `pkg/o11y/fake/fake.go:360-375`, `pkg/o11y/noop/noop_test.go:33-39` e `pkg/o11y/fake/fake_test.go:97-117`.", expected: "A documentação pública deve refletir o contrato observável dos providers `noop` e `fake`.", actual: "O README afirma que `Gauge(...)` retorna `nil` em `noop` e `fake`, mas ambas as implementações retornam erro quando `callback == nil`; isso torna a API pública/documentação ambígua e induz uso incorreto." }`
  Ref de regra: `R-CODE-001` (documentação útil e contrato observável da API pública), `R-TEST-001` (contrato público deve ser coberto e consistente com exemplos/documentação).

### Minor
- Nenhum achado `Minor` adicional com evidência suficiente após `make test`, `make lint` e inspeção manual do pacote.

## Verificação Funcional
- Requisitos verificados: 6/8
- Bugs encontrados: 3
- Evidência objetiva verificada:
  - `make test` passou com `go test -race -coverprofile=coverage.out ./...`, incluindo `pkg/o11y`, `pkg/o11y/otel`, `pkg/o11y/logging`, `pkg/o11y/metrics`, `pkg/o11y/tracing`, `pkg/o11y/noop` e `pkg/o11y/fake`.
  - `make lint` passou com `golangci-lint run --config .github/golangci.yml ./...` -> `0 issues.`
  - A API pública principal tem testes externos em `pkg/o11y/o11y_test.go`, `pkg/o11y/otel_test.go`, `pkg/o11y/example_test.go`, `pkg/o11y/logging/logging_test.go`, `pkg/o11y/metrics/metrics_test.go`, `pkg/o11y/tracing/tracing_test.go`.
  - O pacote evita alterar providers globais do OTel por padrão no provider `otel.NewProvider`; isso é coberto por `pkg/o11y/otel/config_test.go`.
  - O shutdown dos providers principais é idempotente nos testes lidos (`pkg/o11y/otel/config_test.go`, `pkg/o11y/logging/logging_test.go`, `pkg/o11y/metrics/metrics_test.go`, `pkg/o11y/tracing/tracing_test.go`).
  - A sanitização de logs sensíveis e typed-nil errors em `pkg/o11y/otel/logger.go` e `pkg/o11y/otel/error_value.go` tem cobertura objetiva em `pkg/o11y/otel/logger_test.go` e `pkg/o11y/otel/error_value_test.go`.
  - Não ficou objetivamente comprovado que toda configuração inválida falha cedo: `OTLPProtocol` inválido é aceito silenciosamente.
  - Não ficou objetivamente comprovado isolamento de estado global no bootstrap de logging sem exporter: o código mostrou dependência direta de `slog.Default()`.

## Validações Executadas
- `make test` -> sucesso; `go test -race -coverprofile=coverage.out ./...`
- `make lint` -> sucesso; `golangci-lint run --config .github/golangci.yml ./...` retornou `0 issues.`
- `rg --files pkg/o11y` -> mapeamento completo do escopo revisado
- `rg -n '^package ' pkg/o11y --glob '*_test.go'` -> confirmou cobertura com testes externos para APIs públicas relevantes
- `go run /tmp/o11y-invalid-protocol-XXXX.go` -> saída `nil error`, provando aceitação silenciosa de `OTLPProtocol` inválido
- `go run /tmp/o11y-global-logger-XXXX.go` -> escreveu `time=... level=INFO msg=hello` no `slog.Default()` customizado, provando acoplamento ao logger global

## Premissas e Evidências Ausentes
- Não houve diff/PR específico, PRD, TechSpec ou task vinculada. A revisão foi feita sobre o estado atual de `pkg/o11y` no workspace.
- O relatório assume que o contrato alvo é o comportamento observável do pacote publicado em `pkg/o11y` e sua documentação local.

## Riscos Residuais
- Enquanto `OTLPProtocol` inválido continuar sendo aceito silenciosamente, consumidores podem acreditar que estão usando HTTP quando a biblioteca tentará gRPC, gerando falhas de integração difíceis de diagnosticar.
- Enquanto `logging.New` continuar dependente de `slog.Default()` no caminho sem exporter, mudanças globais fora do pacote alteram o comportamento do provider e quebram isolamento entre testes e aplicações.
- Enquanto o README continuar divergente do código para `Gauge(nil)`, integradores podem escrever código incorreto com base na documentação oficial do pacote.
