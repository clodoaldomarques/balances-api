# Especificações do Balances API

Specs geradas por **engenharia reversa** do código em `main` (commit de referência `16ba535`, 2026-11-22). Elas descrevem o comportamento **atual**, não um estado desejado — com exceção da spec 005, que é prescritiva.

## Estrutura

| Arquivo | Conteúdo |
|---|---|
| `001-regras-de-dominio/` | Agregado `Account`, regras de saldo/limite, operações e casos de uso (`internal/domain/accounts`). |
| `002-api-rest-de-contas/` | Rotas Echo, DTOs, validação e envelope de erro (`internal/infra/rest/**`). |
| `003-configuracao-e-infraestrutura/` | `config.New`, persistência MySQL, publicador SNS/SQS, Terraform, k8s, Docker e Makefile. |
| `004-observabilidade-e-shutdown/` | Spans, middleware de log, OTLP, health e desligamento gracioso. |
| `005-hexagonal-adapters/` | Bundle local da spec 005: estado desejado, ainda não implementado neste repo. |

Cada diretório contém `spec.md`, `plan.md` e `tasks.md`.

As specs `001`-`004` são espelhos descritivos do código; a `005` descreve um estado desejado e ainda não implementado. Cada repo da família ledger/balances mantém seu próprio bundle `005-hexagonal-adapters/{spec.md,plan.md,tasks.md}`, com os mesmos IDs e sem depender de links externos.

## Convenção das tarefas

- `[x]` = já implementado e verificável no código em `16ba535`.
- `[ ]` = lacuna identificada ou trabalho a fazer (presente somente na spec 005 e em itens diferidos das specs descritivas).
- `USn` vincula a tarefa a uma história de usuário; `Tnnn` é o marco sequencial; `BA-nnn` é a evidência concreta neste repo.
- `FR-nnn` são requisitos funcionais e `SC-nnn`, critérios de sucesso. A numeração **reinicia em cada spec**; apenas a 005 usa os IDs canônicos da família.
- `AS-IS` identifica comportamento verificado no código; desvios e bugs conhecidos são registrados como **comportamento atual congelado**, não corrigidos nestas specs.

## Uso para estudo

1. Comece pela `001` para entender as regras financeiras e o agregado.
2. Siga para a `002` e a `003` para ver como HTTP, MySQL e SNS expõem e persistem essas regras.
3. Leia a `004` para entender spans, logs e o ciclo de vida do processo.
4. Compare com a `005` para ver o estado arquitetural desejado (hexagonal) e as tarefas pendentes.
5. Se o código mudar, atualize também a spec correspondente — o índice e as specs são espelhos do código.
