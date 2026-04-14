# Padrões de Código

- Rule ID: R-CODE-001
- Severidade: guideline (hard quando ligado a correção, segurança, estabilidade da API pública ou arquitetura)
- Escopo: Todos os arquivos `.go`.

## Objetivo
Garantir código Go idiomático, claro e direto, com nomenclatura consistente, API legível e baixo custo de manutenção para bibliotecas reutilizáveis.

## Referência Mandatória
- O guia base de estilo é o Uber Go Style Guide PT-BR:
  `https://github.com/alcir-junior-caju/uber-go-style-guide-pt-br/blob/main/style.md`
- Os guias mandatórios complementares do projeto são:
  `https://github.com/pedronauck/skills/blob/main/skills/golang-pro/references/interfaces.md`
  `https://github.com/pedronauck/skills/blob/main/skills/golang-pro/references/generics.md`
  `https://github.com/pedronauck/skills/blob/main/skills/golang-pro/references/concurrency.md`
- Na ausência de regra mais específica do projeto, seguir o guia da Uber.
- Para interfaces, generics e concurrency, seguir mandatoriamente os guias complementares acima.

## Requisitos

### Idioma
- Ver Política de Idioma em `governance.md`.
- Símbolos de código, testes, comentários e exemplos devem estar em inglês, salvo termo de domínio que precise permanecer no original.

### Convenções de Nomenclatura
- `camelCase`: variáveis locais, parâmetros de função, campos não exportados.
- `PascalCase`: funções, métodos, structs, interfaces, tipos e constantes exportados.
- `snake_case`: nomes de arquivos e diretórios.
- Nomes de pacote devem ser curtos, lowercase, sem underscore e sem stutter.
- Nomes de interface devem expressar comportamento, sem prefixo `I`.

### Clareza de API
- Funções devem começar com verbo quando representam ação.
- Tipos devem ter nomes de domínio ou responsabilidade, não nomes vagos como `Manager`, `Helper`, `Util` ou `Processor` sem contexto.
- Variáveis booleanas devem ler como asserção (`isActive`, `hasPermission`, `canRetry`).
- Nomes devem privilegiar clareza sobre brevidade extrema.
- Evitar abreviações obscuras; abreviações idiomáticas permitidas: `ctx`, `err`, `id`, `db`, `tx`, `cfg`, `http`.

### Design de Funções
- Preferir guard clauses e early returns.
- Evitar `else` após `return` explícito.
- Evitar condicionais aninhadas com mais de 2 níveis.
- Não usar parâmetros booleanos de flag para alternar comportamento.
- Preferir até 3 parâmetros posicionais; acima disso, considerar `Config`, params struct ou options.
- Métodos exportados devem ser previsíveis, com efeito colateral mínimo e sem "modo mágico".

### API Pública e Documentação
- Todo símbolo exportado deve ter godoc útil, explicando contrato, invariantes e comportamento observável.
- Exemplos de uso devem cobrir o caminho principal da biblioteca quando a API não for óbvia.
- Comentários devem explicar invariantes, trade-offs, protocolos externos ou decisões não triviais.
- Não comentar o óbvio.

### Abstração
- Preferir tipo concreto até que uma interface seja necessária.
- Duplicação pequena e local é preferível a abstração prematura.
- Helpers privados devem ser extraídos apenas quando melhorarem legibilidade, reuso real ou testabilidade.

### Tamanho e Coesão
- Funções: alvo de até 50 linhas.
- Arquivos: alvo de até 300 linhas.
- Pacotes devem manter coesão forte; evitar pacotes "misc", "common", "base" ou "shared" sem fronteira real.

## Proibido
- Símbolos em português no código.
- Parâmetros booleanos de flag que alternam comportamento.
- Stutter do tipo `cache.CacheService`.
- Comentários que apenas restam código óbvio.
- Abstrações criadas apenas para "parecer SOLID".
