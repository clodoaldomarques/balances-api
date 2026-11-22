# Tarefas: `balances-api` — Adoção da Arquitetura Hexagonal

**Repo**: `github.com/clodoaldomarques/balances-api` | **Ordem**: 3 de 4

**Branch**: `005-hexagonal-adapters`

**Status**: Fases 0-3 e 5 concluídas em 2026-11-22; Fase 4 permanece diferida.

**Entrada**: [spec.md](spec.md) e [plan.md](plan.md) deste diretório.

**Nota**: este repo não tinha `docs/specs/` antes da spec 005; as specs descritivas `001`-`004` foram reconstruídas por engenharia reversa em 2026-11-22. Antes da migração, cada handler construía MySQL + SNS + `Service` por request e `Service` morava em `internal/domain/accounts`; após as fases 0-3, a composição está em `cmd/main.go` e o caso de uso em `internal/core/accounts/application`.

**Convenção**: `[ ]` = a fazer; `[x]` = concluído **no código**. Cada checkpoint exige `go test ./...`, `go build ./...` e `go vet ./...` verdes.

**IDs**: `T001`-`T031` são os marcos comuns da família, adaptados ao escopo deste repo; `BA-001`-`BA-022` são os passos concretos que os implementam aqui.

## Fase 0: Baseline e guarda

- [x] T001 Rodar `go test ./...`, `go build ./...` e `go vet ./...` neste repo e registrar o resultado antes da mudança
- [x] T002 [P] Criar `scripts/check-architecture.sh`: falha se `internal/core/**` importar `internal/adapters`, SDK de cloud, Echo ou `config`
- [x] T003 [P] Adicionar alvo `arch-check` ao `Makefile` e documentá-lo neste repo
- [x] T004 Confirmar thread safety da conexão MySQL e do publicador SNS para uso compartilhado entre requests
- [x] BA-001 Rodar `go test ./...`, `go build ./...`, `go vet ./...` e registrar o estado antes da mudança
- [x] BA-002 Criar `scripts/check-architecture.sh` e o alvo `make arch-check`

**Checkpoint**: suíte verde e guarda arquitetural passando no estado atual.

---

## Fase 1: Movimento físico

- [x] T005 [P] Executar o mapeamento de pastas de [plan.md](plan.md) neste repo, sem alterar regras de negócio
- [x] T006 [P] Ajustar todos os imports e a diretiva `go:generate` (que vai junto com `interfaces.go` para `ports/`)
- [x] T007 Rodar `go generate ./...`, `go test ./...` e `go build ./...` após o movimento
- [x] BA-003 `git mv internal/domain/accounts/{account,balances,entry,errors,events,impact,limits,operations}.go → internal/core/accounts/domain/` (manter `account_test.go` junto)
- [x] BA-004 `git mv internal/domain/accounts/service.go → internal/core/accounts/application/service.go` e mover `service_test.go` junto
- [x] BA-005 `git mv internal/domain/accounts/interfaces.go → internal/core/accounts/ports/outbound.go` (com `go:generate`); criar `inbound.go` com `AccountsUseCase`
- [x] BA-006 `git mv internal/domain/accounts/mock.go → internal/core/accounts/ports/mock.go` e regenerar
- [x] BA-007 `git mv internal/infra/rest/accounts → internal/adapters/inbound/http/accounts`; `git mv internal/infra/rest/server → internal/adapters/inbound/http/server`
- [x] BA-008 `git mv internal/infra/database/mysqldb → internal/adapters/outbound/mysql`
- [x] BA-009 `git mv internal/infra/message → internal/adapters/outbound/sns`
- [x] BA-010 Ajustar imports e rodar `go generate ./...` + suíte completa

**Não aplicáveis aqui**: T008 (reparo do build de `balances-worker`); T020-T023 (remoção de `gringotts`/`ProgramAPI`/`ollama`, ausentes neste repo).

**Checkpoint**: `Service` agora em `application` (mudança real de pacote), regras intactas.

---

## Fase 2: Portas explícitas

- [x] T009 [P] Renomear portas de saída para nomes intencionais (`AccountRepository`, `AccountEventPublisher`, `HealthChecker`)
- [x] T010 [P] Declarar a porta de entrada `AccountsUseCase` no core
- [x] T011 [P] Mudar construtores dos adapters para receber conexão/config em vez de chamar `config.New()` internamente
- [x] T012 [P] Regenerar mocks ao lado das portas e ajustar `service_test.go`
- [x] T013 [US1] Criar teste de contrato da porta com fake in-memory executando a mesma suíte do caso de uso
- [x] BA-011 Renomear `Repository` → `AccountRepository`, `Topic` → `AccountEventPublisher`; manter `Event` no core (é contrato de domínio)
- [x] BA-012 Declarar `AccountsUseCase` com os métodos públicos de `Service`
- [x] BA-013 Declarar `HealthChecker` no core e implementá-lo nos adapters MySQL e SNS
- [x] BA-014 Trocar construtores dos adapters para receber conexão/config; remover chamadas internas a `config.New()`
- [x] BA-015 Criar teste de contrato das portas com fake in-memory

**Checkpoint**: core independente de infra.

---

## Fase 3: Composition root e adaptadores de entrada

- [x] T014 [P] Mover a construção de adapters para `cmd/main.go`
- [x] T015 [US2][P] Transformar handlers em structs que recebem a porta de entrada; remover a construção de adapters por request
- [x] T016 [US3][P] Injetar `HealthChecker` no adaptador HTTP
- [x] T017 [US3] Centralizar o mapeador de erros de domínio para status HTTP no adaptador de entrada
- [x] T018 [US3] Garantir que `Close()` dos adapters seja chamado no graceful shutdown
- [x] T019 [US2] Adicionar teste de handler com fake da porta de entrada
- [x] BA-016 Criar `cmd/main.go`: config, conexão MySQL, publicador SNS, `Service`, handlers e servidor
- [x] BA-017 Transformar os handlers em `Handler{uc AccountsUseCase}`; remover a construção de adapters por request
- [x] BA-018 Alterar `server.New()` para receber handlers e `HealthChecker`
- [x] BA-019 Centralizar `400`/`404` em `adapters/inbound/http/errors.go`
- [x] BA-020 Fechar adapters no graceful shutdown e adicionar teste de handler com fake da porta de entrada

**Checkpoint**: nenhuma composição fora da raiz; contrato HTTP inalterado.

---

## Fase 4: Opcional / diferido (fora dos critérios de sucesso)

- [ ] T024 [P] Avaliar decorators para mover `tracer`/`logger` do core para as bordas
- [ ] T025 [P] Avaliar divisão de portas grossas por capacidade
- [ ] T026 Corrigir bugs conhecidos da família em specs próprias, com coordenação de contrato

---

## Fase 5: Documentação e verificação

- [x] T027 [P] Registrar a regra de dependência via portas na documentação de arquitetura deste repo e atualizar versão/data
- [x] T028 [P] Atualizar `docs/specs/README.md` e a documentação afetada com os novos caminhos
- [x] T029 [P] Registrar a convenção arquitetural neste repo
- [x] T030 Rodar `go test ./...`, `go build ./...`, `go vet ./...`, `make arch-check`
- [x] T031 Conferir as respostas das rotas de `accounts` e o contrato SNS; validar SC-001 a SC-007 aplicáveis
- [x] BA-021 Criar `docs/specs/constitution.md` mínima descrevendo a regra de dependência hexagonal e atualizar o índice `docs/specs/README.md` deste repo
- [x] BA-022 Rodar `go test ./...`, `go build ./...`, `go vet ./...`, `make arch-check`

## Dependências e Ordem de Execução

- Fase 0 bloqueia todas as demais.
- Fase 1 deve estar verde antes da Fase 2; a separação de `Service` de `domain` é pré-requisito das portas.
- Fase 2 bloqueia Fase 3.
- `ledger-config` e `ledger-worker` são executados antes deste repo; o padrão é replicado aqui.
- Fase 4 depende de decisão explícita e não é necessária para fechar a spec.
