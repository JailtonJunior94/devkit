# Testes

- Rule ID: R-TEST-001
- Severidade: hard para correção, comportamento público e isolamento; guideline para estilo.
- Escopo: Todos os arquivos `*_test.go`, exemplos executáveis e testes de contrato da API pública.

## Objetivo
Garantir qualidade, previsibilidade e segurança de refatoração para uma biblioteca Go importável por outros projetos.

## Requisitos

### Isolamento e Determinismo
- Testes não devem compartilhar estado mutável.
- Setup por teste deve resetar dependências.
- Testes não devem depender da ordem de execução.
- Testes unitários puros não devem fazer rede, banco real ou acesso a serviços externos.

### Cobertura do Contrato Público
- Toda API pública relevante deve ter teste cobrindo comportamento nominal, erros esperados e casos limite.
- Para bibliotecas, preferir também testes em pacote externo (`package foo_test`) para validar a experiência real de importação.
- Exemplos executáveis (`Example...`) devem ser usados quando melhorarem a documentação e a validação do uso principal.

### Estrutura
- Preferir AAA (Arrange, Act, Assert).
- Usar cenários table-driven para lógica com múltiplas variações.
- Nomes de teste devem descrever o comportamento esperado em inglês.
- Usar `t.Helper()` em helpers de teste.

### Doubles
- Usar mocks, stubs ou fakes apenas quando ajudarem a isolar contrato observável.
- Preferir fake simples a mock complexo quando o comportamento puder ser modelado com menos acoplamento.
- Expectativas de mock devem definir contagem de chamadas quando relevante para o contrato.

### Execução
- O gate padrão deve incluir ao menos `go test ./...`.
- Testes devem ser rápidos o suficiente para rodar frequentemente durante evolução da biblioteca.

## Proibido
- Testes sem asserções.
- Estado mutável compartilhado entre casos.
- Dependência em timing não determinístico sem controle explícito.
- Testar apenas detalhes internos e não o contrato público.
