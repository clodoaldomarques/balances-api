# Tarefas: `balances-api` — Configuração, Persistência MySQL e Mensageria

**Repo**: `github.com/clodoaldomarques/balances-api` | **Ordem**: 3 de 5 (sequência de specs neste repo; a 005 é canônica da família)

**Branch**: `003-configuracao-e-infraestrutura`

**Entrada**: [spec.md](spec.md) e [plan.md](plan.md) deste diretório.

**Nota**: especificação **retroativa**. `[x]` = implementado e verificável no código atual; `[ ]` = pendente. Marcos `T001`-`T015` e evidências `BA-001`-`BA-012`.

**Formato**: `[ID] [P] [US] Descrição com caminho exato do arquivo`.

## Fase 0: Configuração

- [x] T001 [US1] Definir `Config` com todos os campos atuais e getters AWS
- [x] T002 [US1] Implementar `New()` com `sync.Once`, defaults via `core-sdk/env` e aplicação de options
- [x] T003 [US1] Implementar `GetMySQLConnectionString()` com o DSN atual
- [x] BA-001 `config/config.go`: defaults `APP_PORT=5000`, MySQL, AWS e SNS/SQS
- [x] BA-002 Options `WithAppPort`, `WithMysqlDB*`, `WithAwsAddress`, `WithAwsRegion`, `WithSnsAccountTopic`
- [x] BA-003 `.env` local com ARN do tópico `balances-sns-topic` (`8fb2ccc`)

**Checkpoint**: configuração carregada uma vez; DSN idêntico ao usado pela conexão.

---

## Fase 1: Persistência MySQL

- [x] T004 [US2] Implementar `Connect()` com `sql.Open` + `Ping` e log de falha
- [x] T005 [US2][US5] Implementar `Repository` com os quatro métodos de `accounts.Repository`
- [x] T006 [US2] Manter os quatro statements SQL e a chave composta `(account_id, org_id)`
- [x] T007 [US2] Implementar a transação de `SaveEntryAndUpdateAccount`
- [x] T008 [US2] Mapear entidades ↔ tabelas com JSON de limites, saldos e impactos
- [x] BA-004 `internal/infra/database/mysqldb/connection.go` (`8398f3e`)
- [x] BA-005 `internal/infra/database/mysqldb/service.go` (`7245533`, `dd57b8c`)
- [x] BA-006 `internal/infra/database/mysqldb/tables.go` e `commons.DecimalMap`
- [x] BA-007 Tabelas `accounts`, `entries`, `daily_balances` no ConfigMap k8s e no DDL histórico

**Checkpoint**: repositório satisfaz a interface do domínio; esquema aplicado no initContainer.

---

## Fase 2: Mensageria SNS

- [x] T009 [US3][US5] Implementar `Topic.Emit` com `sns.NewPublisher` do `core-sdk`
- [x] T010 [US3] Montar `sns.Event` com `uuid.New()`, `EventType()`, `EventData()` e `time.Now()`
- [x] T011 [US3] Manter `Close(ctx)` com log, sem fechar recursos adicionais
- [x] BA-008 `internal/infra/message/service.go` (`4c2e592`, `d8b5ad3`, `8b5a8cf`)
- [x] BA-009 ARN configurado por `BALANCES_SNS_TOPIC`/`WithSnsAccountTopic`

**Checkpoint**: eventos publicados com envelope e tópico atuais; mocks substituem o adaptador nos testes.

---

## Fase 3: Infraestrutura local

- [x] T012 [US4] Provisionar tópico SNS, política, fila SQS e assinatura com Terraform
- [x] T013 [US4] Criar ConfigMap de schema, Deployment com initContainer e Service 5002 no k8s
- [x] T014 [US4] Definir imagem Docker e alvos do Makefile (`run`, `build`, `push`, `apply`, `destroy`, `terraform`, `test`)
- [x] BA-010 `scripts/terraform/{provider,sns,sqs}.tf` (`866dd08`, `295c681`)
- [x] BA-011 `scripts/k8s/app-{configmap,deployment,service}.yaml` (`ab42c68`, `13c0ff2`)
- [x] BA-012 `scripts/docker/Dockerfile`, `Makefile`, `scripts/sql/ddl_create_database.sql`

**Checkpoint**: `make terraform` e `kubectl apply -f scripts/k8s/` reproduzem o ambiente.

---

## Fase 4: Verificação e fechamento

- [x] T015 Rodar `go build ./...` e `go vet ./...`; conferir SQL, envelope e manifests contra as SCs
- [x] Registrar divergências congeladas: options reaplicadas, conexão por request, `r.ctx`, `Commit` sem erro, `EXPOSE 5000` × 5002

**Checkpoint**: baseline verde em 2026-11-22 (`go build ./...` OK, `go vet ./...` OK).

## Não aplicáveis aqui

- Composição única na raiz e compartilhamento de conexão: spec 005 (FR-004).
- Inspetores de saúde (`HealthChecker`) e graceful shutdown: spec 004 e spec 005.
- Correção de bugs conhecidos (Commit, `r.ctx`, porta): spec futura.

## Dependências e Ordem de Execução

- Depende da spec 001 (interfaces e entidades).
- Anterior à spec 002 (handlers consomem os adapters) e à spec 004 (middleware e ciclo de vida).
- A spec 005 renomeia/move estes adapters sem alterar SQL, envelope SNS nem infraestrutura versionada.
