---
name: refactor
description: Performs safe, incremental refactors that preserve behavior and reduce complexity. Use when simplifying code, isolating hotspots, or improving maintainability. Don't use for bugfix-only requests or review-only audits.
---

# Refactor

<critical>Preservar comportamento e contratos existentes</critical>
<critical>Aplicar passos pequenos, testáveis e reversíveis</critical>

## Modos
- `advisory`: apenas plano e recomendações (padrão)
- `execution`: aplicar refatoração no código

## Definição de Hotspot
Um hotspot é um arquivo ou função que satisfaz qualquer um dos critérios: alta complexidade ciclomática, tamanho excessivo (>50 linhas/função ou >300 linhas/arquivo), violações de regras ou alto acoplamento.

## Entrada
- Receber o objetivo da refatoração e o modo `advisory` ou `execution`.
- Mapear os arquivos e contratos impactados antes de sugerir ou aplicar mudanças.
- Ler `assets/report-template.md` antes de montar a saída final.

## Fluxo de Trabalho
1. Mapear escopo e identificar hotspots usando os critérios acima.
2. Definir objetivo de refatoração por hotspot.
3. Ordenar os hotspots por risco técnico e ganho de simplificação.
4. Aplicar mudanças incrementais apenas no modo `execution`.
5. Validar comportamento com testes e sinais objetivos de preservação de contrato.
6. Relatar mudanças, risco residual e próximos passos.
7. Se o risco aumentar após as mudanças, parar com `blocked`.

## Avaliação de Risco
Risco é determinado por critérios objetivos:
- `Low`: todos os testes passam, sem violações de regras, complexidade mantida ou reduzida. Prosseguir.
- `Medium`: testes passam mas complexidade aumentou ou novas dependências foram introduzidas. Prosseguir com aviso explícito no relatório; chamador decide se aceita.
- `High`: falhas de teste, violações de regras ou contratos quebrados. Parar com `blocked`.

## Persistência de Saída
- Salvar o relatório no caminho indicado pelo chamador.
- Quando invocado no contexto de uma task (`tasks/prd-[feature-name]/`), salvar como `tasks/prd-[feature-name]/refactor_report.md`.
- Sem contexto de task, salvar em `./refactor_report.md`.

## Condições de Parada
- `done`: objetivo do modo selecionado completado com evidência.
- `blocked`: risco residual aumentou ou dependência externa bloqueia progresso.
- `failed`: limite de remediação excedido sem convergência (ver padrão de governança).

## Error Handling
- Se o objetivo da refatoração estiver amplo demais, reduzir para hotspots verificáveis e registrar o corte de escopo.
- Se a mudança alterar API, assinatura pública ou comportamento default sem aprovação explícita, parar com `blocked`.
- Se testes não existirem para a área crítica, registrar a lacuna e elevar o risco residual no relatório.
- Se o modo for `advisory`, não aplicar mudanças e não inferir viabilidade sem evidência suficiente.

## Formato de Saída
- Usar a estrutura de `assets/report-template.md`.
