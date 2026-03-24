# Prompt para iniciar PRD - API de Transfers em Go

```txt
Atue como Product Manager + Tech Lead e gere um PRD inicial (v0.1), em português, para o seguinte objetivo:

Objetivo do produto:
Criar uma pasta `examples/` contendo uma API em Go que use `pkg/o11y` e `pkg/database`, com endpoint `POST /transfers` para um fluxo de transferência “envia para A e recebe em B”.

Fluxo esperado:
1) Receber requisição via API HTTP POST.
2) Persistir a transação no banco com Unit of Work (UoW).
3) Chamar uma API externa de ledger via HTTP para registrar as operações (débito em A e crédito em B).
4) Garantir consistência: ou salva + registra no ledger com sucesso, ou em caso de erro desfaz tudo (rollback/compensação) sem deixar estado inconsistente.

Importante:
- Não assumir 2PC com a API externa de ledger.
- Explicar os limites de rollback com integração HTTP externa.
- Propor a estratégia recomendada para consistência (ex.: transação local + outbox + saga/compensação + idempotência).
- Especificar como lidar com falhas parciais e retries de forma segura.
- Adotar explicitamente melhores práticas de fintech regulada com alta volumetria.

Contexto regulatório e operacional (obrigatório no PRD):
- Considerar requisitos de compliance aplicáveis (ex.: BACEN/CMN, LGPD, PCI DSS quando aplicável, KYC/AML, trilha de auditoria).
- Definir controles de segurança por padrão: criptografia em trânsito e repouso, gestão de segredos, RBAC, segregação de funções, princípio do menor privilégio.
- Definir requisitos de auditabilidade e não repúdio: trilha imutável de eventos críticos, correlação ponta a ponta e retenção de logs.
- Definir requisitos de continuidade: RTO/RPO, estratégia de DR, degradação controlada e plano de incidentes.

Estrutura obrigatória do PRD:
1. Problema e motivação
2. Objetivos de negócio e técnicos
3. Métricas de sucesso (SLI/SLO e métricas de produto)
4. Escopo (in-scope / out-of-scope)
5. Requisitos funcionais
6. Requisitos não funcionais (latência, disponibilidade, segurança, observabilidade, resiliência)
7. Arquitetura proposta (componentes, fronteiras e responsabilidades)
8. Fluxo ponta a ponta:
   - Happy path
   - Fluxos de erro
   - Diagrama textual/sequencial
9. Contrato da API `POST /transfers`:
   - Request/response JSON
   - Códigos HTTP e erros
   - Regras de validação
   - Idempotency-Key
10. Modelo de dados mínimo:
   - Tabela de transações
   - Estados da transação (state machine)
   - Campos de auditoria/correlação
11. Estratégia transacional:
   - Uso de `pkg/database` com Unit of Work
   - Commit/rollback local
   - Publicação/processamento de outbox
   - Chamada ao ledger e compensação
12. Observabilidade com `pkg/o11y`:
   - Logs estruturados
   - Métricas (contadores, latência, taxa de erro)
   - Tracing distribuído
   - Correlation/Trace ID de ponta a ponta
13. Critérios de aceite (Given/When/Then)
14. Estratégia de testes:
   - Unitários
   - Integração (DB + ledger mock/sandbox)
   - Contrato da API
   - Testes de resiliência (timeouts, retry, falha intermitente)
15. Plano de implementação por fases (MVP -> hardening -> produção)
16. Riscos, trade-offs e questões em aberto
17. Compliance e controles regulatórios aplicáveis
18. Operação em alta escala (capacity planning, limites, particionamento, backpressure, filas, rate limit)
19. Runbook operacional (incidentes, reconciliação, reprocessamento, comunicação)

Restrições e critérios adicionais:
- Linguagem: Go.
- Estrutura alvo: `examples/`.
- Incluir pelo menos 5 cenários de falha com comportamento esperado.
- Definir política de retry, timeout, circuit breaker e deduplicação.
- Definir como evitar dupla execução da transferência.
- Definir idempotência forte por chave de negócio e janela temporal.
- Definir estratégia de reconciliação financeira e fechamento operacional (ledger interno x externo).
- Definir estratégia de ordenação e processamento concorrente seguro para evitar race conditions.
- Definir metas explícitas de performance e confiabilidade para alta volumetria (ex.: p95/p99, throughput, error budget).
- Definir padrões de versionamento de API e compatibilidade retroativa.
- Definir política de retenção e mascaramento de dados sensíveis.
- O resultado deve ser acionável para virar backlog técnico.

Formato de saída esperado:
- Markdown claro e direto.
- Tabelas para contratos, estados e cenários de erro.
- Seções curtas com decisões explícitas e justificadas.
```
