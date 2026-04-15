# Changelog

## 1.0.0 (2026-04-15)


### Features

* **database:** add connection pool manager, unit of work and migrate modules ([029a601](https://github.com/JailtonJunior94/devkit/commit/029a601018c6245e60301a4dab651de833595715))
* **event:** add thread-safe event dispatcher with handler registry and context support ([9eb6f7f](https://github.com/JailtonJunior94/devkit/commit/9eb6f7f50ab1b5d0fd787360fb3febd13bda8b91))
* **messaging:** add agnostic consumer interface and Kafka implementation ([a745f29](https://github.com/JailtonJunior94/devkit/commit/a745f295d756c5a8bdc915407178744ed77ee6d9))
* **messaging:** add RabbitMQ producer and consumer support ([20fc1b6](https://github.com/JailtonJunior94/devkit/commit/20fc1b667c19c7ff6e01e4fc251a37405df44629))
* **o11y:** add per-signal OTLP adapters, noop package and expand test coverage ([f40eeae](https://github.com/JailtonJunior94/devkit/commit/f40eeaeb80428fb484dbf87b7d3341f300277aba))
* **o11y:** implement OpenTelemetry observability library with logging, metrics, tracing and OTLP adapters ([52416dd](https://github.com/JailtonJunior94/devkit/commit/52416ddf288ce2c07f6a66071bb5f25f9d836d01))
* **oteltest:** add TracerProvider accessor and make Shutdown idempotent ([75e992f](https://github.com/JailtonJunior94/devkit/commit/75e992fd6d424737074d49835ba1f0f9e539e664))
* **worker:** add worker/job manager with cron scheduling and graceful shutdown ([f23e363](https://github.com/JailtonJunior94/devkit/commit/f23e3631211c884db32ad37d0e5b75f0dcda17e2))
