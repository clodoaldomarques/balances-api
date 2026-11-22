# Tarefas: `balances-api` — API REST de Contas e Lançamentos

**Repo**: `github.com/clodoaldomarques/balances-api` | **Ordem**: 2 de 5 (sequência de specs neste repo; a 005 é canônica da família)

**Branch**: `002-api-rest-de-contas`

**Entrada**: [spec.md](spec.md) e [plan.md](plan.md) deste diretório.

**Nota**: especificação **retroativa**. `[x]` = implementado e verificável no código atual; `[ ]` = pendente. Marcos `T001`-`T014` e evidências `BA-001`-`BA-009` seguem a convenção da família.

**Formato**: `[ID] [P] [US] Descrição com caminho exato do arquivo`.

## Fase 0: Servidor e rotas

- [x] T001 Criar o `Server` sobre Echo com `Start`, `Shutdown` e registro central de rotas
- [x] T002 [US6] Registrar `GET /` com o corpo de health atual
- [x] T003 [US1][US2][US3][US4] Registrar as quatro rotas de contas
- [x] BA-001 `internal/infra/rest/server/server.go`: `New()`, `routes()`, `Start()`, `Shutdown()`, `HealthCheck`
- [x] BA-002 Rotas em `server.go` — ordem idêntica à da spec 002

**Checkpoint**: servidor compila e expõe exatamente cinco rotas.

---

## Fase 1: DTOs e validação

- [x] T004 [US1] Definir `PostAccountRequest`, `ToEntity()` e `PostAccountResponse`
- [x] T005 [US2][US3] Definir `PutAccountLimitsRequest`, `PutAccountStatusRequest` e `PutAccountResponse`
- [x] T006 [US4] Definir `PostEntryRequest`, `PostImpactRequest` e o conversor de impactos
- [x] T007 [US5] Implementar `CustomValidator` encadeando `validator.Struct` e `EntityRequest.Validate`
- [x] BA-003 `types.go`: listas `validLimits`, `validBalances`, `validOperations`, `validStatus`, `validRules`
- [x] BA-004 `Validate()` das quatro requisições e conversões `AccountTo*Response`
- [x] BA-005 `CustomValidator` instalado como `s.http.Validator` em `server.go`

**Checkpoint**: payload inválido é rejeitado antes de chegar ao domínio.

---

## Fase 2: Handlers e contrato de erro

- [x] T008 [US1] Implementar `CreateNewAccount` — bind, validação, caso de uso, `201`
- [x] T009 [US2] Implementar `UpdateAccountLimits` — parse de `accountID`, `200`
- [x] T010 [US3] Implementar `UpdateAccountStatus` — conversão de `Status`, `200`
- [x] T011 [US4] Implementar `ProcessEntry` — `200` com saldo atualizado
- [x] T012 [US5] Implementar `ErrorResponse` com `ERR0001`/`ERR0002`/`ERR0003`
- [x] BA-006 Quatro handlers em `internal/infra/rest/accounts/handler.go`
- [x] BA-007 `ErrorResponse` em `types.go`; uso sempre com `http.StatusBadRequest`

**Checkpoint**: status e corpo de erro idênticos para as quatro rotas.

---

## Fase 3: Integração e inicialização

- [x] T013 [US1][US4] Conectar handlers a `mysqldb.NewRepository`, `message.NewTopic` e `accounts.NewService`, com `defer Close`
- [x] T014 Executar `cmd/main.go` com `config.WithAppPort(5002)` e `server.Start(c.AppPort)`
- [x] BA-008 Construção de adapters por request em `handler.go` (estado atual, a migrar na spec 005)
- [x] BA-009 `cmd/main.go` iniciando o servidor em goroutine (`13c0ff2`, `8fb2ccc`)

**Checkpoint**: API responde após o start; compose/k8s usam a porta 5002.

---

## Verificação e fechamento

- [x] Rodar `go build ./...` e `go vet ./...` — OK em 2026-11-22
- [x] Conferir rotas, DTOs e códigos de erro contra a tabela de [plan.md](plan.md)
- [x] Registrar a divergência de portas (Docker `EXPOSE 5000` × Service/targetPort `5002`) como comportamento atual

## Não aplicáveis aqui

- Testes de handler com fake da porta de entrada: spec 005 (T019).
- Health readiness com `HealthChecker` injetado: spec 005 (FR-006).
- Movimento dos handlers para `internal/adapters/inbound/http`: spec 005 (Fase 1).

## Dependências e Ordem de Execução

- Depende da spec 001 para entidades, erros e regras de domínio.
- Depende da spec 003 para a conexão MySQL e o publicador SNS.
- A spec 004 instala o middleware de log no `Server` desta spec.
- A spec 005 preserva integralmente o contrato documentado aqui (SC-005/FR-010).
