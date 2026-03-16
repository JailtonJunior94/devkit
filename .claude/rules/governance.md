# Governança de Regras

- Rule ID: R-GOV-001
- Severidade: hard
- Escopo: Todos os arquivos em `.claude/rules/`, `.claude/commands/` e `.claude/skills/`.

## Objetivo
Definir governança determinística para regras, commands e skills: precedência, severidade, estrutura de prompt, resolução de conflitos, segurança de execução e política de evidência.

## Escopo das Regras
Todos os commands e skills seguem implicitamente todas as regras em `.claude/rules/`. Regras têm precedência em caso de conflito. Todos os commands e skills têm acesso implícito aos arquivos de `.claude/context/`.

## Metadados de Regra
Cada arquivo de regra deve declarar:
- `Rule ID`: identificador único.
- `Severidade`: `hard` ou `guideline`.
- `Escopo`: arquivos, camadas ou artefatos afetados.

## Precedência
1. `governance.md`
2. `security.md`
3. `architecture.md`
4. `error-handling.md`
5. `tests.md`
6. `code-standards.md`

Se duas regras do mesmo nível conflitarem:
1. Preferir a regra com maior severidade (`hard` > `guideline`).
2. Se mesma severidade, preferir o comportamento mais restritivo para segurança, correção e estabilidade da API pública.

## Modelo de Severidade

### hard
Bloqueante para merge. Não pode ser ignorada.
Aplica-se a: segurança, dados sensíveis, exposição de erros, fronteiras de arquitetura, estabilidade da API pública, comportamento determinístico do agente.

### guideline
Não bloqueante por padrão. Pode ser ignorada com justificativa documentada.
Aplica-se a: nomenclatura, metas de tamanho, estilo de teste, ergonomia da API e convenções de documentação.

## Máquina de Estados Canônica
- Estados de execução permitidos: `pending`, `in_progress`, `needs_input`, `blocked`, `failed`, `done`.
- Vereditos de gate permitidos: `APPROVED`, `APPROVED_WITH_REMARKS`, `REJECTED`, `BLOCKED`.
- Estados e vereditos são enums separados; nunca misturar.

## Práticas de Prompt para Commands e Skills
- Instruções devem ser claras, diretas e específicas sobre tarefa, contexto, restrições e resultado esperado.
- Sempre explicitar o objetivo da tarefa, o que caracteriza sucesso e as restrições não negociáveis.
- Para workflows multi-etapa, usar passos sequenciais numerados.
- Quando o prompt tiver múltiplas partes, estruturar com tags XML ou blocos claramente nomeados, por exemplo: `<context>`, `<requirements>`, `<constraints>`, `<acceptance_criteria>`, `<examples>`.
- Em tarefas com muito contexto, posicionar documentos-fonte antes da instrução final e preservar metadados de origem quando relevante.
- Quando a resposta depender de evidência textual, exigir grounding em trechos objetivos do contexto fornecido.
- Se informação obrigatória estiver ausente ou incerta, o agente deve declarar a incerteza explicitamente e parar com `needs_input` em vez de inventar resposta.
- Evitar contexto irrelevante, redundante ou sensível que não seja necessário para executar a tarefa.

## Restrições de Segurança do Agente
- Todo command ou skill deve declarar condições de parada explícitas.
- Workflows de longa duração devem definir ciclos máximos de remediação.
- Se input obrigatório estiver ausente, a execução deve parar com status `needs_input`.
- Operações destrutivas requerem intenção explícita do usuário na thread atual.
- Se a intenção for ambígua, parar com `needs_input`.
- Em dependências externas indisponíveis, continuar apenas com suposições explícitas registradas no relatório de execução.
- Progresso baseado em suposições não pode aprovar gates de segurança, correção ou compatibilidade.

## Política de Evidência
- Relatórios de execução devem incluir: comandos executados, arquivos alterados, resultados de validação, suposições e riscos residuais.
- Decisões de gate devem incluir: nome do gate, veredito e razão objetiva.
- Aprovação exige evidência observável.
- Termos vagos como "provavelmente ok" ou "parece bom" não são evidência suficiente.

## Limite de Remediação Padrão
- Salvo quando sobrescrito, o máximo de ciclos de remediação padrão é **2** por estágio ou por bug.

## Política de Idioma
- Símbolos de código, testes, comentários de código e exemplos devem estar em inglês.
- Texto operacional em rules, commands e skills pode estar em português quando isso melhorar clareza para o time.
- Termos legais ou de produto podem preservar a nomenclatura original quando necessário.

## Proibido
- Loops infinitos de remediação sem limite.
- Transições de estado implícitas.
- Concluir tarefas com achados críticos ou breaking changes não tratados.
- Nomes de estado ad-hoc fora dos enums canônicos.
- Aprovação sem evidência.
- Prompt confuso, contraditório ou com instruções escondidas em texto solto quando a estrutura puder ser explícita.
