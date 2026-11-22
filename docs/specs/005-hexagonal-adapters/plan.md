# Plano de Implementação: Adoção da Arquitetura Hexagonal — `balances-api`

**Branch**: `005-hexagonal-adapters` | **Data**: 2026-11-22 | **Repo**: `github.com/clodoaldomarques/balances-api`

**Entrada**: [spec.md](spec.md) deste diretório e a estrutura atual deste repo.

**Nota**: este plano descreve trabalho **a executar** neste repo. `[x]` só deve ser usado após a implementação real. Todo o conteúdo necessário está aqui e nos arquivos vizinhos deste diretório.

## Resumo

Separar o pacote único `internal/domain/accounts` em `internal/core/accounts/{domain,application,ports}`, mover handlers e adapters para `internal/adapters/{inbound,outbound}` e transformar `cmd/main.go` na raiz de composição. `Service` sai de `domain` e passa a ser caso de uso em `application`; portas `Repository`/`Topic` viram `AccountRepository`/`AccountEventPublisher`, com `AccountsUseCase` e `HealthChecker` no core. Regras de saldo/limites, contrato HTTP e contrato SNS ficam congelados.

## Contexto Técnico

**Linguagem/Versão**: Go (ver `go.mod` deste repo)

**Dependências principais**: Echo v4, MySQL driver, AWS SDK v2 (SNS), `core-sdk` (tracer/logger/métricas), `go.uber.org/mock` (mockgen), `shopspring/decimal` (valores monetários), OpenTelemetry

**Armazenamento**: MySQL (database `balances` por padrão via `WithMysqlDBName`); nenhuma alteração de schema

**Testes**: `go test ./...`, `go build ./...`, `go vet ./...`; `account_test.go` e `service_test.go` hoje vivem junto das entidades e devem acompanhar o movimento

**Mocks**: `go generate` a partir de `interfaces.go`; `mock.go` é gerado e nunca editado à mão

**Plataforma alvo**: serviço HTTP em container (porta 5002 por padrão)

**Restrições**: sem CI, sem autenticação, `config.New()` é singleton `sync.Once`, tracing dentro do core permanece nesta fase, sem collection Postman versionada neste repo.

**Escopo neste repo**: 1 separação real de pacote (`Service` de `domain` para `application`), 4 portas, 2 adaptadores de saída, 1 adaptador de entrada HTTP, 1 guarda de arquitetura, documentação mínima de specs.

## Verificação da Constituição

*GATE: aplicado antes da Fase 0 e reavaliado ao fim da Fase 6.*

| Princípio | Situação planejada | Evidência alvo neste repo |
|---|---|---|
| I. Separação em camadas | **Reescrito** para "regra de dependência via portas" | `internal/core/accounts/**` sem imports de adapters/MySQL/AWS/Echo/config; raiz em `cmd/main.go` |
| II. O domínio valida, a interface transporta | Passa | DTOs em `adapters/inbound/http/accounts`; regras de conta/entry/limites em `domain` |
| III. Configuração por ambiente, com singleton explícito | Passa (inalterado) | `config.New()` continua `sync.Once`; opções aplicadas uma vez na raiz |
| IV. Observabilidade é parte do request | Passa (com desvio documentado) | Spans permanecem no caso de uso; decorators ficam para spec futura |
| V. Consultas são desenhadas no modelo de dados | Passa | Consultas e tabelas MySQL intactas |
| VI. Testes unitários determinísticos | Passa | Mocks ao lado das portas; fake de `AccountsUseCase` no teste de handler |
| VII. Infraestrutura como código | Passa | Compose/Terraform/k8s sem mudança; `make arch-check` adicionado |

**Ação obrigatória na Fase 6**: registrar a regra de dependência via portas na documentação de arquitetura deste repo (que hoje é mínima) e atualizar sua data/versão.

## Estrutura do Projeto

### Documentação (esta funcionalidade, neste repo)

```text
docs/specs/005-hexagonal-adapters/
├── spec.md      # US1-US5, FR-001..FR-014, SC-001..SC-007 aplicados a balances-api
├── plan.md      # este documento: layout, mapeamento, riscos, verificação
└── tasks.md     # fases 0-6 com IDs T001-T031 e BA-001-BA-022
```

### Código-fonte alvo

```text
cmd/main.go                              # composition root
config/config.go                         # mantido
internal/
├── core/accounts/
│   ├── domain/                          # account.go, balances.go, entry.go, errors.go,
│   │                                    # events.go, impact.go, limits.go, operations.go
│   ├── application/                     # service.go, service_test.go
│   └── ports/
│       ├── inbound.go                   # AccountsUseCase
│       ├── outbound.go                  # AccountRepository, AccountEventPublisher, HealthChecker (+ go:generate)
│       └── mock.go                      # gerado
└── adapters/
    ├── inbound/http/
    │   ├── accounts/                    # handlers + DTOs
    │   ├── server/                      # Echo, rotas, health
    │   └── errors.go                    # mapeador 400/404
    └── outbound/
        ├── mysql/                       # AccountRepository + HealthChecker
        └── sns/                         # AccountEventPublisher + HealthChecker
scripts/check-architecture.sh            # guarda de imports
```

**Decisão de estrutura**: manter `Event`/`EventType()`/`EventData()` no domínio (são contrato de negócio) e mover apenas o `Service` para `application`. Essa é a única mudança real de pacote além dos adapters.

## Mapeamento de Pastas

| Origem | Destino | Observações |
|---|---|---|
| `internal/domain/accounts/{account,balances,entry,errors,events,impact,limits,operations}.go` | `internal/core/accounts/domain/` | `account_test.go` acompanha |
| `internal/domain/accounts/service.go` | `internal/core/accounts/application/service.go` | `service_test.go` acompanha; mudança real de pacote |
| `internal/domain/accounts/interfaces.go` | `internal/core/accounts/ports/outbound.go` | `go:generate` vai junto; criar `inbound.go` com `AccountsUseCase` |
| `internal/domain/accounts/mock.go` | `internal/core/accounts/ports/mock.go` | Regenerar a partir das portas finais |
| `internal/infra/rest/accounts` | `internal/adapters/inbound/http/accounts` | Handlers viram structs com `AccountsUseCase` |
| `internal/infra/rest/server` | `internal/adapters/inbound/http/server` | `server.New()` passa a receber handlers e `HealthChecker` |
| `internal/infra/database/mysqldb` | `internal/adapters/outbound/mysql` | Conexão injetada pela raiz |
| `internal/infra/message` | `internal/adapters/outbound/sns` | Publicador injetado pela raiz |

## Riscos e Decisões

| Risco / desvio | Por que existe | Decisão |
|---|---|---|
| `Service` hoje mora em `domain` | Pacote mistura entidades e caso de uso | Separar em `application` na Fase 1; ajustar imports e testes |
| `config.New(...)` dentro da goroutine do servidor | Configuração tardia e repetida | Aplicar opções uma vez em `cmd/main.go` |
| Handlers constroem MySQL + SNS + `Service` por request | Composição espalhada | Injetar portas por construtor na Fase 3 |
| `Event` é interface de domínio | Contrato de negócio | Permanece no core; não virar DTO de adaptador |
| `tracer`/`logger` dentro do core | Já presentes | Manter e registrar como desvio |
| Ausência de specs `001`-`004` e de collection Postman | Repo não tinha esse histórico documentado | Specs `001`-`004` reconstruídas por engenharia reversa em 2026-11-22; verificar contrato com testes e rotas atuais |
| Mover diretórios quebra imports de uma vez | Mudança física | Fase 1 exclusivamente `git mv` + imports, com build verde antes de seguir |
| Bugs conhecidos da família | Foram registrados como comportamento atual | Congelar; cada correção em spec própria |

## Estratégia de Migração

1. **Fase 0 — Baseline**: suíte verde e `make arch-check` criado.
2. **Fase 1 — Movimento físico**: `git mv` + imports + `go generate`, separando `Service` de `domain`; nenhuma mudança de regra.
3. **Fase 2 — Portas**: renomear `Repository`/`Topic`, declarar `AccountsUseCase` e `HealthChecker`, adapters recebem conexão/config.
4. **Fase 3 — Composition root**: `cmd/main.go` compõe tudo; handlers viram structs; erros passam pelo mapper.
5. **Fase 4 — Opcional (fora do SC)**: decorators de tracing, divisão de portas, correções de bugs conhecidos.
6. **Fase 5 — Fechamento**: documentação mínima de specs, verificações finais.

Cada fase DEVE terminar com build e testes verdes. Rollback por commit de fase; nunca misturar fases no mesmo commit.

## Verificação

```bash
# executar na raiz do repo

go test ./...
go build ./...
go vet ./...
make arch-check

# após o movimento e a separação de Service
go generate ./internal/core/accounts/...
go test ./internal/core/accounts/... ./internal/adapters/...
go mod tidy

# rastreabilidade dos nomes antigos e do Service em domain
grep -R -E 'infra/(rest|database|message)|domain/accounts/service' --include='*.go' . || true
```

## Dependências e Ordem de Execução

- A spec deve ser aprovada antes da Fase 0.
- `ledger-config` e `ledger-worker` são migrados antes deste repo; o padrão é replicado aqui.
- Fase 0 bloqueia todas as demais; Fase 1 deve estar verde antes da Fase 2.
- Fase 2 bloqueia Fase 3; a inicialização de config na raiz é pré-requisito da Fase 3.
- Fase 4 depende de decisão explícita e não é necessária para fechar a spec.
