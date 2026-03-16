# Caminhos do Projeto

- **API pública da biblioteca:** raiz do módulo ou `pkg/{lib}/`
- **Detalhes internos não importáveis:** `internal/`
- **Adapters opcionais:** subpacotes específicos por integração, sem contaminar o núcleo da biblioteca
- **Regras:** `.claude/rules/`
- **Commands:** `.claude/commands/`
- **Skills:** `.claude/skills/`
- **Templates:** `.claude/templates/`
