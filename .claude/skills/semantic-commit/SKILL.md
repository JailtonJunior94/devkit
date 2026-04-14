---
name: semantic-commit
description: Generates Conventional Commit messages from a git diff and can propose commit splitting or a short PR summary. Use when deriving a semantic commit from staged or unstaged changes. Don't use for bugfix, refactor, or review execution.
---

# Semantic Commit

<critical>Inferir tipo de commit a partir de evidência do diff</critical>
<critical>Usar formato Conventional Commit</critical>

## Formato do Commit
`<type>(scope-opcional): <descrição>`

## Tipos Permitidos
`feat`, `fix`, `refactor`, `perf`, `docs`, `test`, `chore`, `build`, `ci`, `style`

## Entrada
- Receber um diff legível, staged ou unstaged.
- Ler `assets/output-template.md` antes de montar a resposta final.

## Fluxo de Trabalho
1. Analisar diff e agrupar mudanças por intenção.
2. Inferir tipo e escopo.
3. Gerar mensagem de commit principal.
4. Se existirem mudanças não relacionadas, sugerir divisão em commits separados.
5. Opcional: gerar resumo curto de PR.

## Regras de Desempate
- Múltiplas intenções: priorizar `feat` > `fix` > `refactor` > `perf` > `docs` > `test` > `chore` > `build` > `ci` > `style`.
- Mudanças independentes sem objetivo dominante: sugerir divisão (obrigatório).

## Condições de Parada
- `done`: commit semântico (e opcionalmente divisão/resumo) gerado a partir do diff.
- `needs_input`: diff ausente ou ilegível.

## Error Handling
- Se o diff contiver mudanças sem relação clara entre si, não forçar um único commit; sugerir divisão obrigatória.
- Se o escopo não puder ser inferido com segurança, omitir o `scope` em vez de inventá-lo.
- Se houver apenas mudanças mecânicas de formatação, priorizar `style` ou `chore` conforme a evidência do diff.
- Se o diff estiver ausente, truncado ou ilegível, retornar `needs_input`.

## Formato de Saída
- Usar a estrutura de `assets/output-template.md`.
