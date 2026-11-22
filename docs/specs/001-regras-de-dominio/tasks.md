# Tarefas: `balances-api` — Regras de Domínio de Contas, Saldos e Lançamentos

**Repo**: `github.com/clodoaldomarques/balances-api` | **Ordem**: 1 de 5 (sequência de specs neste repo; a 005 é canônica da família)

**Branch**: `001-regras-de-dominio`

**Entrada**: [spec.md](spec.md) e [plan.md](plan.md) deste diretório.

**Nota**: especificação **retroativa**. `[x]` = implementado e verificável no código atual; `[ ]` = pendente (não há pendências nesta spec). Os marcos `T001`-`T014` são a sequência histórica; `BA-001`-`BA-010` são a evidência concreta neste repo.

**Convenção**: cada checkpoint foi validado com `go test ./internal/domain/accounts/...`, `go build ./...` e `go vet ./...`.

**Formato**: `[ID] [P] [US] Descrição com caminho exato do arquivo`.

## Fase 0: Baseline do modelo

- [x] T001 Definir o agregado `Account` com saldos/limites em `decimal.Decimal`, status e versão
- [x] T002 Definir as constantes de status, saldo, limite, operação e regra
- [x] BA-001 `internal/domain/accounts/account.go`, `balances.go`, `limits.go`, `operations.go`, `entry.go`, `impact.go`
- [x] BA-002 `docs/model/03 - C4 - Domain.puml` descreve `Account`, `Entries` e `Impact`

**Checkpoint**: modelo compilando e alinhado ao diagrama de domínio.

---

## Fase 1: Regras de negócio

- [x] T003 [US1] Implementar `ChangeStatus` e `IncreaseVersion`
- [x] T004 [US3] Implementar `ChangeLimit` com validações de max/total/overdraft e conta inativa
- [x] T005 [US2] Implementar `ChangeBalances` com débito/crédito, regras de saldo e rejeição de crédito com regras
- [x] T006 [US2] Implementar validação de saldo disponível considerando `total_limit`
- [x] BA-003 `ChangeLimit`, `ChangeBalances`, `IncreaseVersion` em `account.go`
- [x] BA-004 `validateDebit*Balance` em `balances.go`; `validateChange*Limit` em `limits.go`; `creditFun`/`debitFun` em `operations.go`

**Checkpoint**: `TestAccount_ChangeStatus`, `TestAccount_ChangeLimit` e `TestAccount_ChangeBalances` verdes.

---

## Fase 2: Erros de domínio

- [x] T007 [US2][US3] Padronizar os quatro erros tipados e suas mensagens
- [x] BA-005 `ErrInsuficientBalance`, `ErrAccountDisabled`, `ErrValidateLimit`, `ErrValidateOperation` em `errors.go` (`ae352f8`)

**Checkpoint**: mensagens de erro idênticas às asserções dos testes.

---

## Fase 3: Casos de uso e portas

- [x] T008 [US4] Implementar `Service` com os quatro casos de uso, persistência antes da emissão
- [x] T009 [US4] Declarar `Repository`, `Topic` e `Event` e gerar os mocks
- [x] T010 [US4] Instrumentar os casos de uso com spans e log de erro
- [x] BA-006 `NewService`, `CreateNewAccount`, `UpdateAccountLimits`, `UpdateAccountStatus`, `ProcessEntry` em `service.go`
- [x] BA-007 `interfaces.go` com `//go:generate mockgen ...` e `mock.go` gerado
- [x] BA-008 `tracer.NewSpanFromContext`/`SetError` e `logger.Error` em `service.go`

**Checkpoint**: `TestService*`/`TestNewService` verdes; falha de repositório impede `Emit`.

---

## Fase 4: Contratos de evento

- [x] T011 [US5] Definir os três tipos de evento e seus payloads JSON
- [x] T012 [US5] Construir os eventos a partir das entidades nos casos de uso
- [x] BA-009 `events.go`: `create_account`, `update_account`, `process_entry`, `ImpactEvent` (`4c2e592`)
- [x] BA-010 `buildCreateAccountEvent`, `buildUpdateAccountEvent`, `buildProcessEntryEvent`, `buildImpactEvents` em `service.go`

**Checkpoint**: `EventType()`/`EventData()` e campos JSON congelados.

---

## Fase 5: Verificação e fechamento

- [x] T013 Rodar `go test ./internal/domain/accounts/...`, `go build ./...` e `go vet ./...`
- [x] T014 Registrar nesta spec os desvios que não devem ser corrigidos junto da spec 005

**Checkpoint**: baseline verde em 2026-11-22 — `go test ./...` OK (pacote `internal/domain/accounts`), `go build ./...` OK, `go vet ./...` OK.

## Não aplicáveis aqui

- Persistência, publicação SNS, configuração e HTTP: specs 003 e 002.
- Movimento de `Service` para `application` e renomeação de portas: spec 005.
- Correção de grafias, validação de overdraft, tratamento do retorno de `Emit`: spec futura.

## Dependências e Ordem de Execução

- Fase 0 bloqueia a Fase 1; a Fase 1 bloqueia a Fase 3 (casos de uso dependem das regras).
- A Fase 4 depende da Fase 3 (eventos são construídos após a persistência).
- A spec 002 consome esta; a spec 003 persiste/publica estas entidades; a spec 004 instrumenta o `Service`.
