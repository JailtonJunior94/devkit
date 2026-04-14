# Reviewer Decision Framework

## Política de Decisão
- `REJECTED`: qualquer achado `Critical` não resolvido ou qualquer violação de regra `hard`.
- `APPROVED_WITH_REMARKS`: sem `Critical` ou `Major` não resolvidos, com apenas itens `Minor` residuais que não invalidam requisito.
- `APPROVED`: sem achados não resolvidos e sem requisito atendido apenas por inferência.
- `BLOCKED`: evidência ou inputs obrigatórios ausentes, diff incompleto, impossibilidade de executar validações mínimas ou requisito sem prova objetiva suficiente para veredito determinístico.

## Rubrica de Severidade
- `Critical`: viola regra `hard`, quebra requisito mandatório, cria risco de perda de dados, corrupção, panic, falha de segurança, regressão funcional evidente ou comportamento incorreto em produção sem mitigação aceitável.
- `Major`: requisito importante não comprovado, bug relevante com workaround ruim, arquitetura incompatível com PRD ou TechSpec, cobertura insuficiente em área crítica, ou documentação e API pública ambíguas a ponto de induzir uso incorreto.
- `Minor`: melhoria localizada, clareza, documentação, teste complementar ou risco baixo sem impacto funcional ou material imediato.
- Em caso de dúvida entre duas severidades, escolher a maior até existir evidência contrária.
