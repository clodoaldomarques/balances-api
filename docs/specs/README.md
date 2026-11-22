# Especificações do Balances API

Specs geradas por **engenharia reversa** do código em `main` (commit de referência `16ba535`, 2026-11-22), com exceção da spec 005, que é prescritiva e foi **implementada** em 2026-11-22.

## Estrutura

| Arquivo | Conteúdo |
|---|---|
| `001-regras-de-dominio/` | Agregado `Account`, regras de saldo/limite, operações e caso de uso. Caminhos históricos: `internal/domain/accounts`. |
| `002-api-rest-de-contas/` | Rotas Echo, DTOs, validação e envelope de erro. Caminhos históricos: `internal/infra/rest/**`. |
| `003-configuracao-e-infraestrutura/` | `config.New`, persistência MySQL, publicador SNS/SQS, Terraform, k8s, Docker e Makefile. |
| `004-observabilidade-e-shutdown/` | Spans, middleware de log, OTLP, health e desligamento gracioso. |
| `005-hexagonal-adapters/` | Bundle da spec 005: estado desejado, plano e tarefas executadas. Layout atual: `internal/core/**` e `internal/adapters/**`. |
| [`constitution.md`](constitution.md) | Regra de dependência hexagonal adotada e desvios registrados. |

Cada diretório de spec contém `spec.md`, `plan.md` e `tasks.md`.

## Convenção das tarefas

- `[x]` = implementado e verificável no código.
- `[ ]` = trabalho diferido (atualmente apenas a Fase 4 da spec 005).
- `USn` vincula a tarefa a uma história de usuário; `Tnnn` é o marco sequencial; `BA-nnn` é a evidência concreta neste repo.
- `FR-nnn` são requisitos funcionais e `SC-nnn`, critérios de sucesso. A numeração **reinicia em cada spec**; apenas a 005 usa os IDs canônicos da família.
- `AS-IS` identifica comportamento verificado no código em `16ba535`; desvios e bugs conhecidos são registrados como **comportamento atual congelado**, não corrigidos nestas specs.

## Migração para o layout hexagonal

| Caminho histórico (`16ba535`) | Caminho atual (spec 005) |
|---|---|
| `internal/domain/accounts/{account,balances,entry,errors,events,impact,limits,operations}.go` | `internal/core/accounts/domain/` |
| `internal/domain/accounts/service.go` | `internal/core/accounts/application/service.go` |
| `internal/domain/accounts/interfaces.go` | `internal/core/accounts/ports/{inbound,outbound}.go` |
| `internal/domain/accounts/mock.go` | `internal/core/accounts/ports/mock.go` |
| `internal/infra/rest/accounts` | `internal/adapters/inbound/http/accounts` |
| `internal/infra/rest/server` | `internal/adapters/inbound/http/server` |
| `internal/infra/database/mysqldb` | `internal/adapters/outbound/mysql` |
| `internal/infra/message` | `internal/adapters/outbound/sns` |

As specs `001`-`004` mantêm os caminhos históricos como evidência do estado `AS-IS`. Para o layout vigente, use a tabela acima, a spec 005 e a [`constitution.md`](constitution.md). Detalhes operacionais estão em [`../platform.md`](../platform.md).

## Uso para estudo

1. Comece pela `001` para entender as regras financeiras e o agregado.
2. Siga para a `002` e a `003` para ver como HTTP, MySQL e SNS expõem e persistem essas regras.
3. Leia a `004` para entender spans, logs e o ciclo de vida do processo.
4. Leia a `005` para ver a migração hexagonal executada e a regra de dependência via portas.
5. Consulte a `constitution.md` e a `platform.md` para o estado atual do código e da plataforma.
