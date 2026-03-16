# Documento de Requisitos do Produto (PRD) - Esteira de CI/CD (GitHub Actions)

## Visão Geral

Este documento descreve os requisitos para a implementação de uma esteira de Integração Contínua (CI) e Entrega Contínua (CD) utilizando GitHub Actions para o projeto `devkit`. O objetivo é automatizar a validação de código, garantir padrões de qualidade, segurança e automatizar o versionamento semântico da biblioteca.

## Objetivos

- **Qualidade de Código**: Garantir que todo código submetido siga os padrões definidos pelo projeto através de linting rigoroso.
- **Integridade**: Validar que novas alterações não quebrem funcionalidades existentes através de testes de unidade automatizados.
- **Segurança**: Identificar proativamente vulnerabilidades conhecidas e práticas inseguras de codificação.
- **Automação de Release**: Automatizar a criação de tags de versão baseadas em Commits Semânticos, facilitando o consumo por outros projetos.

## Histórias de Usuário

- **Como um Desenvolvedor**, quero que meu código seja analisado automaticamente em cada Pull Request para que eu receba feedback rápido sobre erros de lint ou falhas em testes.
- **Como um Mantenedor**, quero que o sistema gere automaticamente uma nova versão (tag) sempre que um código for mesclado na branch principal, baseando-se no impacto das mudanças (fix, feat, breaking change).
- **Como um Analista de Segurança**, quero que a esteira verifique vulnerabilidades em dependências e no código fonte para garantir que a biblioteca seja segura para uso.

## Funcionalidades Core

### 1. Análise de Linter (Lint Step)
- **O que faz**: Executa análise estática de código.
- **Ferramenta**: `golangci-lint` (conforme https://golangci-lint.run/).
- **Requisito**: Deve falhar a build se houver violações das regras configuradas.

### 2. Testes de Unidade (Test Step)
- **O que faz**: Executa a suíte de testes do projeto.
- **Ferramenta**: `go test`.
- **Requisito**: Deve executar todos os pacotes (`./...`) e reportar cobertura de código. A build deve falhar se qualquer teste falhar.

### 3. Verificação de Segurança (Security Step)
- **O que faz**: Varre o código em busca de vulnerabilidades.
- **Ferramentas sugeridas**: `govulncheck` (oficial do Go) e/ou `gosec`.
- **Requisito**: Identificar vulnerabilidades em dependências (SCA) e no código fonte (SAST).

### 4. Versionamento Semântico Automatizado (Release/Tag Step)
- **O que faz**: Analisa os commits desde a última tag e gera uma nova tag semântica (v1.x.x).
- **Lógica**: Seguir a especificação de Semantic Versioning (SemVer) baseada em Conventional Commits.
- **Requisito**: 
    - `fix:` -> gera patch release.
    - `feat:` -> gera minor release.
    - `BREAKING CHANGE:` ou `feat!:` -> gera major release.
- **Output**: Uma nova tag git criada automaticamente no repositório após o merge na branch principal.

## Restrições Técnicas de Alto Nível

- **Plataforma**: GitHub Actions.
- **Linguagem**: Go (versão atual do projeto definida em `go.mod`).
- **Padrão de Commit**: Obrigatoriedade de Commits Semânticos para que o versionamento funcione corretamente.
- **Segurança**: Segredos (tokens) devem ser gerenciados via GitHub Secrets.
- **Performance**: A esteira deve ser otimizada utilizando cache de dependências do Go e do golangci-lint.

## Fora de Escopo

- Deploy em ambientes de staging ou produção (foco apenas em biblioteca/SDK).
- Geração automática de CHANGELOG detalhado nesta fase inicial (opcional, mas desejável futuramente).
- Testes de integração complexos que exijam infraestrutura externa (ex: bancos de dados reais).
