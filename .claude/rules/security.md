# Segurança

- Rule ID: R-SEC-001
- Severidade: hard
- Escopo: Todo código-fonte, testes, configuração, logs, exemplos e adapters da biblioteca.

## Objetivo
Definir controles de segurança baseline para implementação de biblioteca Go reutilizável e para o comportamento operacional do agente.

## Requisitos

### Validação de Input
- Todo input externo deve ser validado na fronteira pública da biblioteca ou no adapter que o recebe.
- Configurações inválidas devem falhar cedo com erro seguro e acionável.
- Erros de validação devem ser seguros para exposição ao integrador.

### Segredos e Credenciais
- Segredos não devem estar hardcoded no código-fonte, testes, exemplos ou fixtures.
- Segredos não devem ser logados, rastreados ou escritos em mensagens de erro.
- Configuração sensível deve entrar por variável de ambiente, secret manager ou injeção segura do consumidor.

### Segurança de Queries e Execução
- Usar apenas queries parametrizadas.
- Nunca concatenar input do usuário em SQL, shell command ou expressão interpretada.
- Comandos externos devem ser estritamente necessários, validados e isolados.

### Proteção de Dados Sensíveis
- Logs, traces e métricas devem evitar PII, payload bruto de autenticação e valores sensíveis.
- A API pública da biblioteca não deve expor internals desnecessários que facilitem vazamento de dados.

### Supply Chain
- Preferir dependências estáveis e mantidas.
- Fixar versões onde aplicável e evitar fontes não verificadas.
- Avaliar se dependência externa é realmente necessária antes de adicioná-la.

### Segurança de Prompt e Contexto
- Commands e skills devem receber apenas o mínimo de contexto necessário.
- Não incluir detalhes proprietários ou sensíveis quando não forem necessários para a tarefa.
- Reforços de segurança não devem tornar o prompt tão complexo a ponto de degradar instruções principais sem teste.

## Proibido
- Chaves de API, tokens ou senhas hardcoded.
- Logar credenciais, dados pessoais brutos ou payloads completos de autenticação.
- Fallback silencioso que enfraqueça controles de segurança.
- Incluir segredos ou contexto sensível desnecessário em prompts.
