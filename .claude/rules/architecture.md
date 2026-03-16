# Arquitetura de Biblioteca Go

- Rule ID: R-ARCH-001
- Severidade: hard
- Escopo: Todo código-fonte Go da biblioteca, com foco em pacotes exportados na raiz do módulo ou em `pkg/`, e detalhes não exportáveis em `internal/`.

## Objetivo
Garantir design Go idiomático para uma biblioteca reutilizável, fácil de importar, com API pública estável, baixo acoplamento, fronteiras claras e abstrações introduzidas apenas quando resolvem um problema real.

## Requisitos

### Topologia de Pacotes
- Código importável por outros projetos deve viver em pacote exportado na raiz do módulo ou em `pkg/{lib}`.
- Detalhes que não devem ser importados externamente devem viver em `internal/`.
- Adaptadores opcionais para frameworks, banco, mensageria ou HTTP devem ficar em pacotes separados para não contaminar a API principal.
- Evitar estrutura orientada a camada sem necessidade; preferir organização por capacidade da biblioteca.

### API Pública
- A superfície pública deve ser pequena, explícita e orientada ao caso de uso principal da biblioteca.
- Símbolos exportados devem existir apenas quando forem necessários para integração externa.
- Toda função ou método exportado que faça I/O, bloqueie ou dependa de cancelamento deve aceitar `context.Context` como primeiro parâmetro.
- Construtores exportados devem usar `New...`.
- Quando configuração crescer além de poucos campos, usar `Config` ou Option Pattern em vez de listas longas de parâmetros.
- Zero value deve ser útil quando seguro; quando não for, o construtor deve deixar a obrigatoriedade explícita.

### Dependências e Acoplamento
- Dependências devem apontar para dentro: pacotes opcionais/adapters -> núcleo da biblioteca.
- O núcleo da biblioteca não deve importar adapters, transportes ou frameworks consumidores.
- **Dependências externas são permitidas e esperadas**: esta é uma biblioteca de integração — seu propósito é encapsular e simplificar o uso de SDKs e frameworks externos. Adicionar dependências externas é legítimo quando a biblioteca entrega valor justamente por integrar essas deps.
- A decisão entre stdlib e dependência externa deve considerar: a dep já é a abstração correta para o caso de uso? Reimplementar seria pior? Se sim, usar a dep.
- Dependências externas devem ser estáveis, bem mantidas e com escopo claro. Evitar deps que trazem acoplamento desnecessário ao *núcleo* quando poderiam ficar em sub-pacotes opcionais.
- Deps que aumentam significativamente a superfície transitiva de um consumidor (ex.: stack gRPC completo) devem ficar em sub-pacotes opcionais para que consumidores que não as precisem não as carreguem.

### Interfaces
- Interfaces devem ser pequenas, comportamentais e definidas no pacote consumidor da abstração.
- Não retornar interface em construtor por padrão; retornar tipo concreto quando há uma implementação principal e a abstração não agrega valor.
- Introduzir interface apenas para desacoplamento real entre pacotes, extensão por plugin, mocking ou múltiplas implementações relevantes.
- Evitar interfaces "espelho" de structs concretas.

### Padrões de Projeto
- Aplicar padrão de projeto apenas para resolver uma pressão concreta de design.
- Padrões aceitáveis quando justificados: Strategy para comportamento variável, Factory para construção validada, Adapter para integração externa, Decorator para cross-cutting concerns.
- O código deve registrar no comentário ou na documentação curta qual problema o padrão resolve.
- Evitar Singleton, Service Locator, Abstract Factory ou hierarquias genéricas de "manager/provider/service" sem necessidade demonstrável.

### Modelagem e Invariantes
- Tipos de domínio e configuração devem proteger invariantes na criação e mutação.
- Value objects devem se autovalidar e permanecer imutáveis por design quando fizer sentido.
- Regras de negócio ou de consistência da biblioteca não devem vazar para adapters.
- Inputs externos devem ser convertidos cedo para tipos seguros do domínio da biblioteca.

### Erros e Observabilidade
- O núcleo da biblioteca deve devolver erros; logging e telemetria devem ser opt-in e injetáveis.
- Nunca exigir logger global, tracer global ou estado global mutável para a biblioteca funcionar.
- Recursos com `Close() error` devem ter cleanup explícito com captura do erro.
- Erros de infraestrutura não devem ser convertidos para semântica de transporte dentro do núcleo da biblioteca.

### Evolução e Compatibilidade
- Mudanças breaking na API pública exigem mudança explícita de major version.
- Pacotes exportados devem preservar nomes, contratos e semântica com o menor churn possível.
- Helpers internos não devem ser exportados "por conveniência".

## Proibido
- Forçar estrutura de monólito de aplicação em biblioteca reutilizável.
- Acoplar o núcleo a framework HTTP, ORM, driver específico ou CLI.
- Estado global mutável para fluxo principal da biblioteca.
- Dependências circulares.
- Interfaces vazias ou genéricas demais sem contrato claro.
- Expor detalhes internos de adapter na API pública.
