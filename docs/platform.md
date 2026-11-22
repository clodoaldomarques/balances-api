# Plataforma e Composição — `balances-api`

**Versão**: 1.0 | **Atualizado em**: 2026-11-22

Este documento descreve a plataforma após a spec `005-hexagonal-adapters`. A regra de dependência está registrada em [`specs/constitution.md`](specs/constitution.md).

## Composição raiz

`cmd/main.go` executa, nesta ordem:

1. `config.New(...)` uma única vez, com `WithAppPort(5002)` e `WithMysqlDBName("balances")`;
2. `mysql.Connect(dsn)` e `mysql.NewRepository(ctx, db)`;
3. `sns.NewPublisher(ctx, cfg)` com o publicador do `core-sdk`;
4. `application.NewService(repository, publisher)`;
5. `accounts.New(useCase)` e `server.New(handler, healthCheckers{repository, publisher})`;
6. `Start`, `otel.Start`, espera por sinal e graceful shutdown com `Shutdown`, `Close` dos adapters e `otel.Shutdown`.

Nenhum handler constrói adapters ou casos de uso.

## Health check

A rota `GET /` continua respondendo:

```json
{"data": "Server is up and running"}
```

O adaptador HTTP consulta a porta `HealthChecker` injetada antes de responder. A raiz compõe um checker que consulta MySQL (`PingContext`) e SNS (`Health`). O status HTTP e o corpo permanecem os mesmos da versão anterior.

## Segurança concorrente

- `database/sql.DB` é um pool seguro para uso concorrente; uma única instância é compartilhada pela vida do processo.
- O cliente AWS SDK v2 usado pelo publicador SNS é reutilizável entre requisições.
- A transação em `SaveEntryAndUpdateAccount` continua usando o `ctx` capturado no construtor, comportamento congelado pela spec `003`.

## Guarda arquitetural

```bash
make arch-check
```

O script `scripts/check-architecture.sh` falha quando:

- `internal/core/**` importa `internal/adapters`, infra legada, cloud SDK, Echo, `config`, `database/sql` ou o driver MySQL;
- construtores de adapters/caso de uso são chamados fora de `cmd/**` e dos próprios adapters.

## Verificação local

```bash
go test ./...
go build ./...
go vet ./...
make arch-check
```
