# Prompt para Criação da PRD: Implementação do RabbitMQ em pkg/messaging

Você é um Product Manager Sênior técnico. Seu objetivo é escrever um Documento de Requisitos do Produto (PRD) detalhado para a implementação de um novo driver de mensageria utilizando **RabbitMQ** no pacote `pkg/messaging`.

## Contexto do Projeto
O projeto já possui uma abstração de mensageria em `pkg/messaging/consumer.go` e uma implementação robusta para Kafka em `pkg/messaging/kafka`. A nova implementação do RabbitMQ deve seguir os mesmos padrões de design e oferecer funcionalidades equivalentes.

## Requisitos para a PRD
A PRD deve seguir o template oficial em `.claude/templates/prd-template.md` e incluir os seguintes pontos:

1.  **Visão Geral:** Explicar a necessidade de suportar RabbitMQ como uma alternativa leve e flexível ao Kafka para diferentes casos de uso de mensageria no `devkit`.
2.  **Objetivos:**
    *   Implementar o contrato `messaging.Consumer` definido em `pkg/messaging/consumer.go`.
    *   Garantir paridade de funcionalidades com a implementação Kafka (retries, DLQ, concorrência via workers).
    *   Prover uma configuração flexível baseada em `Options`.
3.  **Histórias de Usuário:** Focar em desenvolvedores que utilizam o `devkit` e precisam integrar seus microserviços via RabbitMQ.
4.  **Funcionalidades Core:**
    *   Conexão e Reconexão Automática (AMQP 0-9-1).
    *   Suporte a diferentes tipos de Exchange (Direct, Topic, Fanout).
    *   Consumo de filas com suporte a ACK/NACK.
    *   Mecanismo de Retry com backoff exponencial.
    *   Suporte a Dead Letter Exchange (DLX) / Dead Letter Queue (DLQ).
    *   Processamento paralelo utilizando worker pools (similar ao Kafka).
    *   Graceful Shutdown.
5.  **Restrições Técnicas:**
    *   Utilizar a biblioteca padrão de mercado para Go (ex: `github.com/rabbitmq/amqp091-go`).
    *   Seguir a documentação oficial: https://www.rabbitmq.com/.
    *   Manter a compatibilidade com o logger (`slog`) e telemetria do projeto.
6.  **Fora de Escopo:** Implementação de Produtor (se o foco for apenas Consumer no momento, ou definir se o Producer também deve ser incluído agora).

## Instruções Adicionais
*   Use um tom profissional e técnico.
*   Garanta que a estrutura do documento facilite a criação subsequente da Tech Spec.
*   Considere os padrões de erro já estabelecidos no projeto.

---
**Gere a PRD agora com base nestas diretrizes.**
