# Plano de Implementação: Configuração, Persistência MySQL e Mensageria — `balances-api`

**Branch**: `003-configuracao-e-infraestrutura` | **Data**: 2026-11-22 | **Spec**: [spec.md](spec.md)

**Repo**: `github.com/clodoaldomarques/balances-api`

**Entrada**: [spec.md](spec.md) deste diretório, `config/config.go`, `internal/infra/database/mysqldb/**`, `internal/infra/message/service.go` e `scripts/**`.

**Nota**: plano **retroativo**. Descreve as decisões já implementadas; `[x]` em [tasks.md](tasks.md) significa "implementado e verificável no código".

## Resumo

Centralizar a configuração em `config.New()` (env + singleton + options), implementar o repositório `accounts.Repository` com `database/sql`/MySQL e um schema de três tabelas, implementar `accounts.Topic` com o publicador SNS do `core-sdk` e um envelope com uuid, e provisionar tópico SNS, fila SQS, schema k8s, container e comandos Makefile. Os adapters dependem do domínio, nunca o contrário.

## Contexto Técnico

**Linguagem/Versão**: Go (módulo `go 1.26.1`; baseline com Go 1.27.1)

**Dependências principais**: `core-sdk` (`env`, `aws/sns`, `commons.DecimalMap`, `zap/logger`), `go-sql-driver/mysql` v1.10.x, AWS SDK v2 (indireto, via core-sdk), `google/uuid`

**Armazenamento**: MySQL — database `balances`; tabelas `accounts`, `entries`, `daily_balances`; colunas JSON para limites, saldos e impactos

**Testes**: nenhum teste de integração neste repo; os testes de domínio usam mocks e não exigem MySQL/AWS

**Mocks**: `Repository` e `Topic` gerados em `internal/domain/accounts/mock.go`

**Plataforma alvo**: Minikube + LocalStack (`192.168.49.2:30002` / `192.168.67.2:30002` no Makefile), Kubernetes com LoadBalancer na porta 5002

**Restrições**: SQL, envelope SNS, ARN do tópico e defaults de config congelados; sem CI

**Escopo neste repo**: 1 arquivo de config, 3 arquivos MySQL, 1 adaptador SNS, 4 arquivos Terraform, 3 manifests k8s, 1 SQL, 1 Dockerfile, 1 Makefile

## Verificação da Constituição

| Princípio | Situação no código | Evidência |
|---|---|---|
| I. Separação em camadas | Passa — adapters importam domínio e config, nunca o inverso | `mysqldb/service.go`, `message/service.go` |
| II. O domínio valida, a interface transporta | Passa | JSON de limites/saldos mapeado em `tables.go`; regras em `domain` |
| III. Configuração por ambiente, com singleton explícito | **Desvio registrado** — options aplicadas a cada `New()` | `config/config.go`; raiz corrige na spec 005 |
| IV. Observabilidade é parte do request | Passa parcialmente — adapters usam `logger` | `connection.go`, `service.go`, `message/service.go` |
| V. Consultas são desenhadas no modelo de dados | Passa | statements em `mysqldb/service.go`, colunas JSON no ConfigMap |
| VI. Testes unitários determinísticos | Passa — mocks substituem adapters | `service_test.go` |
| VII. Infraestrutura como código | Passa | Terraform + k8s + Dockerfile + Makefile |

## Estrutura do Projeto

```text
config/config.go                                  # Config, Option, sync.Once, getters, DSN
internal/infra/database/mysqldb/
├── connection.go                                 # Connect() → sql.Open + Ping
├── service.go                                    # Repository + 4 SQL + transação
└── tables.go                                     # mapeamento entidade ↔ linha/JSON
internal/infra/message/service.go                 # Topic.Emit → sns.Event
scripts/terraform/{provider,sns,sqs}.tf           # provider LocalStack, tópico, fila, assinatura
scripts/k8s/{app-configmap,app-deployment,app-service}.yaml
scripts/sql/ddl_create_database.sql               # DDL histórico
scripts/docker/Dockerfile                         # imagem Go + Alpine
Makefile                                          # run/build/push/apply/destroy/terraform/test
.env                                              # valores locais (não versionado)
```

**Decisão de persistência**: saldos e limites são gravados como JSON em colunas `limits`/`balances`, e impactos como JSON em `entries.impacts`. Isso preserva o formato de mapas do domínio e evita tabelas normalizadas; o custo é não poder consultar valores indivíduos por SQL — aceito porque o worker faz o lado de leitura.

## Mapeamento de Interfaces

| Contrato (`interfaces.go`) | Implementação | Observações |
|---|---|---|
| `SaveNewAccount` | `mysqldb.Repository.SaveNewAccount` | `INSERT_ACCOUNT` |
| `UpdateExistingAccount` | `mysqldb.Repository.UpdateExistingAccount` | `UPDATE_ACCOUNT` |
| `RetrieveAccountByID` | `mysqldb.Repository.RetrieveAccountByID` | `SELECT_ACCOUNT` por `(account_id, org_id)` |
| `SaveEntryAndUpdateAccount` | `mysqldb.Repository.SaveEntryAndUpdateAccount` | transação com `INSERT_ENTRIES` + `UPDATE_ACCOUNT` |
| `Emit` | `message.Topic.Emit` | `sns.Event` com uuid e `time.Now()` |

## Complexidade Conhecida

| Desvio | Por que existe | Decisão (alternativa considerada) |
|---|---|---|
| Options aplicadas em toda chamada de `New()` | `sync.Once` protege só os defaults | Congelar; spec 005 exige aplicação única na raiz |
| Conexão por request | `NewRepository` chama `Connect()` | Congelar; spec 005 propõe compartilhar conexão e validar thread safety |
| `r.ctx` capturado no construtor | Implementação histórica | Congelar; não usar `ctx` do método nesta spec |
| `tx.Commit()` sem checagem de erro | Implementação histórica | Congelar; correção em spec própria |
| `Topic.Close` apenas loga | Publisher do core-sdk não expõe close | Manter |
| Dois endpoints de LocalStack no repo | Evolução do ambiente | Documentar; não unificar |
| `EXPOSE 5000` × Service 5002 | Evolução da configuração de porta | Documentar como divergência atual |

## Evolução Observada (histórica)

1. `bd4411e`-`866dd08` — projeto inicial, Docker, Terraform e LocalStack.
2. `5093334`-`8398f3e` — refatoração da persistência e migração para MySQL (com DynamoDB ainda presente).
3. `4c2e592`-`d8b5ad3` — eventos de domínio e remoção do publisher próprio em favor do `core-sdk`.
4. `d738d60`-`eab8468` — configuração lida uma vez e configuração final.
5. `8fb2ccc`-`ab42c68` — variáveis de ambiente, portas e criação automática do schema no k8s.

## Verificação

```bash
# executar na raiz do repo
go build ./...
go vet ./...
grep -R 'INSERT_ACCOUNT\|SELECT_ACCOUNT\|UPDATE_ACCOUNT\|INSERT_ENTRIES' internal/infra/database/mysqldb/service.go
grep -R 'balances-sns-topic\|balances-sqs-queue' scripts/terraform scripts/k8s .env
```

## Dependências e Ordem de Execução

- Depende da spec 001 para os contratos `Repository`, `Topic`, `Event` e para as entidades mapeadas.
- A spec 002 consome estes adapters por meio dos handlers.
- A spec 004 instrumenta `Service` e gerencia o ciclo de vida do processo; esta spec não altera esse fluxo.
- A spec 005 move os adapters para `internal/adapters/outbound`, mantendo SQL, envelope e infra inalterados.
