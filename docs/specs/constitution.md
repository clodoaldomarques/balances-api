# Convenção Arquitetural — `balances-api`

**Versão**: 1.0 | **Data**: 2026-11-22 | **Origem**: spec `005-hexagonal-adapters`

## Regra de dependência

O código de negócio vive em `internal/core/accounts/**` e **não depende** de adaptadores, infraestrutura, SDK de nuvem, Echo, `config` nem de `database/sql`.

A dependência aponta sempre **do adaptador para o core**:

```text
cmd/main.go (composition root)
        │
        ├── internal/adapters/inbound/http/**  ──► ports.AccountsUseCase
        │                                      ──► ports.HealthChecker
        │
        ├── internal/adapters/outbound/mysql   ──► ports.AccountRepository
        │                                      ──► ports.HealthChecker
        │
        └── internal/adapters/outbound/sns     ──► ports.AccountEventPublisher
                                               ──► ports.HealthChecker

internal/core/accounts/application ──► ports + domain
internal/core/accounts/ports       ──► domain
internal/core/accounts/domain      ──► (somente bibliotecas de domínio)
```

## Portas

| Porta | Tipo | Definição |
|---|---|---|
| `AccountsUseCase` | entrada | `internal/core/accounts/ports/inbound.go` |
| `AccountRepository` | saída | `internal/core/accounts/ports/outbound.go` |
| `AccountEventPublisher` | saída | `internal/core/accounts/ports/outbound.go` |
| `HealthChecker` | saída | `internal/core/accounts/ports/outbound.go` |

`Event` permanece no domínio: é contrato de negócio, não DTO de adaptador.

## Composição

`cmd/main.go` é o único lugar que conhece adapters concretos e `config`. Ele cria config, conexão, publicador, caso de uso, handlers e servidor uma única vez, e fecha os adapters no graceful shutdown. Handlers não chamam construtores de adapter nem `NewService`.

## Validação

| Comando | Objetivo |
|---|---|
| `make arch-check` | falha com arquivo e linha se o core importar pacotes proibidos ou se houver composição fora de `cmd/` e dos adapters |
| `go test ./...` | suíte de domínio, caso de uso, contrato com fake in-memory e handlers com fake da porta |
| `go build ./...` / `go vet ./...` | compilação e análise estática |

## Desvios registrados

- `tracer` e `logger` permanecem dentro do caso de uso nesta spec; decorators nas bordas são Fase 4.
- Comportamentos congelados das specs `001`-`004` (incluindo `r.ctx` na transação MySQL e o `Commit` sem tratamento de erro) não foram corrigidos aqui.
- O contrato HTTP, o envelope SNS e o schema MySQL permanecem inalterados.
