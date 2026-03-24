# Prompt para iniciar o PRD: Módulo de Database (Devkit)

Crie um PRD completo para um módulo de banco de dados em Go para o `devkit`, focado em flexibilidade, padrões de projeto e desacoplamento. O objetivo é ter um `Database Manager` central que suporte múltiplos drivers e funcionalidades opcionais como Migrations e Unit of Work (UOW).

Referências principais:
- https://refactoring.guru/design-patterns (Strategy, Factory, Facade, Proxy)
- https://github.com/golang-migrate/migrate
- https://github.com/JailtonJunior94/devkit-go/tree/main/pkg/database/uow
- https://dev.to/gretro/unit-of-work-pattern-in-go-i6l

## Objetivo do Produto

Desenvolver um componente de persistência robusto que abstraia a complexidade de conexão, gestão de pool e transações, permitindo:
- Suporte nativo a Postgres, MySQL e SQL Server.
- Gestão centralizada de ciclo de vida (inicialização e Graceful Shutdown).
- Integração opcional de Migrações (Up/Down) sem acoplamento rígido.
- Implementação do padrão Unit of Work (UOW) para transações complexas, utilizável de forma independente ou integrada.

## Requisitos Obrigatórios

O PRD deve detalhar:

1. **Database Manager:** Interface única para obter instâncias de conexão, mas permitindo acesso ao driver nativo (`*sql.DB`) se necessário.
2. **Multi-DB Support:** Uso do padrão `Strategy` ou `Abstract Factory` para trocar de engine (Postgres, MySQL, SQL Server) apenas via configuração.
3. **Connection Pool:** Configuração exposta para MaxOpenConns, MaxIdleConns, ConnMaxLifetime, etc.
4. **Graceful Shutdown:** Método `Close()` ou similar que garanta o encerramento seguro das conexões.
5. **Decoupled Migration:** 
   - Integração com `golang-migrate`.
   - Deve ser possível rodar migrações no startup ou via comando separado.
   - O `dbmanager` NÃO deve depender do pacote de migração para funcionar.
6. **Unit of Work (UOW):**
   - Implementação do padrão para gerenciar transações em múltiplos repositórios.
   - O `UOW` deve ser agnóstico ao `dbmanager`, podendo ser injetado ou usado separadamente.
   - Suporte a transações aninhadas ou gestão de erro/rollback automático.
7. **Independência de Uso:**
   - Cenário A: Uso apenas do `dbmanager` para queries simples.
   - Cenário B: Uso do `dbmanager` + `UOW` para lógica de negócio complexa.
   - Cenário C: Uso do `dbmanager` + `Migrate` no deploy/startup.
   - Cenário D: Uso de todos os componentes juntos.

## Diretrizes de Arquitetura

- **Design Patterns:** Identificar onde aplicar `Singleton` (para o manager, se aplicável), `Factory` (para criação de drivers), `Strategy` (para dialetos SQL) e `Decorator/Proxy` (para o UOW).
- **Interface Segregation:** Interfaces pequenas e específicas para que o consumidor não precise implementar o que não usa.
- **Dependency Injection:** Tudo deve ser injetável para facilitar testes unitários.
- **Error Handling:** Padronização de erros de banco (Unique Constraint, Not Found, etc.) para evitar vazamento de tipos do driver para a camada de domínio.

## Expectativas para a API (Exemplos Conceituais)

O PRD deve propor assinaturas para:
- `database.New(config) (Manager, error)`
- `manager.GetDB() *sql.DB`
- `migration.New(manager, fs) (Migrator, error)` -> Note que o migrador recebe o manager.
- `uow.New(manager) (UnitOfWork, error)`
- Exemplos de como o `UOW` registra repositórios e executa o `Do(ctx, fn)`.

## Pontos Técnicos que o PRD deve responder

- Como o `UOW` manterá o estado da transação no `context.Context` de forma segura?
- Como gerenciar diferentes sintaxes de migração (scripts SQL) para diferentes bancos no mesmo migrador?
- Como expor métricas de pool de conexão (usando o módulo de o11y do devkit)?
- Como garantir que o `Close()` do manager também finalize transações pendentes no `UOW`?

## Formato de Saída

Responda em Markdown estruturado, incluindo:
- Arquitetura de pacotes (ex: `pkg/database`, `pkg/database/postgres`, `pkg/database/uow`, `pkg/database/migrate`).
- Tabela de comparação entre os drivers suportados.
- Guia de inicialização rápida para os diferentes cenários de uso.
- Lista de dependências externas permitidas.

## Tom Esperado

- Altamente técnico e pragmático.
- Focado em extensibilidade (facilitar a adição de um 4º banco como Oracle futuramente).
- Sem "over-engineering": a abstração deve simplificar o uso, não esconder o poder do SQL.
