# Tratamento de Erros

- Rule ID: R-ERR-001
- Severidade: hard
- Escopo: Todos os arquivos `.go` com criação, wrapping, tratamento, classificação ou exposição de erros.

## Objetivo
Garantir propagação segura e previsível de erros em uma biblioteca Go, preservando contexto técnico para o integrador sem acoplar o núcleo a transportes específicos.

## Requisitos

### Definição de Erros
- Use erros sentinela apenas quando o chamador realmente precisar tomar decisão com `errors.Is`.
- Use erros tipados apenas quando houver payload semântico necessário para o chamador.
- Nomes de erros exportados devem usar prefixo `Err`.
- Mensagens de erro devem ser lowercase, concisas e estáveis.

### Wrapping e Propagação
- Usar `fmt.Errorf(... %w ...)` para preservar cadeia de erro.
- Usar `errors.Is` e `errors.As` para inspeção.
- Nunca comparar erro com `==`, exceto `nil`.
- A camada mais próxima do problema deve adicionar o contexto técnico útil; camadas acima não devem repetir wrap redundante.
- Funções exportadas devem retornar erros acionáveis, sem esconder a causa real.

### Biblioteca Reutilizável
- O núcleo da biblioteca não deve mapear erro para HTTP, gRPC, CLI exit code ou outro transporte.
- Mapeamento para transporte pertence ao projeto consumidor ou a adapter explícito.
- Se a biblioteca expõe erro sentinela ou tipado, a documentação da API deve indicar quando ele pode ocorrer.
- Logging de erro dentro da biblioteca deve ser opt-in; evitar logar e retornar o mesmo erro por padrão.

### Segurança
- Nunca expor segredos, credenciais, tokens, PII ou detalhes internos sensíveis em mensagens de erro.
- Erros destinados ao usuário final devem ser produzidos pela camada de integração, não pelo núcleo técnico.

## Proibido
- Engolir erros silenciosamente.
- Converter erro técnico em `nil`.
- Usar `panic` para erro recuperável.
- Acoplar erro do núcleo a semântica HTTP ou framework específico.
- Expor detalhes internos sensíveis em mensagens de erro.
