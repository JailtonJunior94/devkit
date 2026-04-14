---
name: bugfix
description: Executes root-cause bug fixes with regression tests and validation evidence. Use when fixing documented defects or reviewer findings. Don't use for code review-only tasks or refactor-only requests.
---

# Bugfix

<critical>Todo bug corrigido deve ter um teste de regressão</critical>
<critical>Não finalizar com bugs pendentes no escopo acordado</critical>

## Entrada
- Receber uma lista de bugs no formato canônico `{ id, severity, file, line, reproduction, expected, actual }`.
- Receber o escopo aprovado dos bugs a corrigir.
- Ler `assets/report-template.md` antes de redigir o relatório final.

## Fluxo de Trabalho
1. Ler e priorizar os bugs por severidade, impacto e facilidade de reprodução.
2. Confirmar a reprodução ou reunir evidência objetiva suficiente para localizar a causa raiz.
3. Corrigir um bug por vez para evitar misturar evidências e regressões.
4. Adicionar ou ajustar um teste de regressão para cada bug corrigido.
5. Executar validações relevantes, priorizando `make test` e `make lint` quando disponíveis.
6. Registrar para cada item a causa raiz, a correção aplicada, os testes executados e a evidência de validação.
7. Parar com `needs_input` se dados obrigatórios de reprodução, escopo ou ambiente estiverem ausentes.

## Persistência de Saída
- Salvar o relatório no caminho indicado pelo chamador.
- Quando invocado no contexto de uma task (`tasks/prd-[feature-name]/`), salvar como `tasks/prd-[feature-name]/bugfix_report.md`.
- Sem contexto de task, salvar em `./bugfix_report.md`.

## Condições de Parada
- `done`: escopo acordado corrigido e validado.
- `blocked`: bug crítico depende de contexto externo não resolvido.
- `needs_input`: dados obrigatórios de reprodução/escopo ausentes.
- `failed`: limite de remediação excedido (ver padrão de governança).

## Error Handling
- Se o bug não puder ser reproduzido, registrar a lacuna de evidência, listar os passos tentados e retornar `blocked` ou `needs_input`.
- Se a correção proposta introduzir regressão, reverter a abordagem localmente, documentar o risco e tentar uma alternativa menor.
- Se `make test` ou `make lint` não existirem, executar a validação automatizada equivalente disponível e registrar o comando usado.
- Se houver múltiplos bugs com a mesma causa raiz, consolidar a explicação, mas manter teste e status por bug.

## Formato de Saída
- Usar a estrutura de `assets/report-template.md`.
