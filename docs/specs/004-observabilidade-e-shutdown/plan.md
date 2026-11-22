# Plano de Implementação: Observabilidade, Health e Shutdown — `balances-api`

**Branch**: `004-observabilidade-e-shutdown` | **Data**: 2026-11-22 | **Spec**: [spec.md](spec.md)

**Repo**: `github.com/clodoaldomarques/balances-api`

**Entrada**: [spec.md](spec.md) deste diretório, `cmd/main.go`, `internal/domain/accounts/service.go`, `internal/infra/rest/server/server.go` e `scripts/k8s/app-deployment.yaml`.

**Nota**: plano **retroativo** — descreve decisões **já presentes** no código; não é um plano a executar do zero. `[x]` em [tasks.md](tasks.md) significa "implementado e verificável no código".

## Resumo

Instrumentar os quatro casos de uso com spans nomeados (`Service::*`), instalar o interceptador de log do `core-sdk` no Echo com redação de campos sensíveis, iniciar/encerrar o OpenTelemetry no `main` e tratar `SIGINT`/`SIGTERM` com shutdown gracioso de 25 segundos na ordem HTTP → otel. A rota `GET /` permanece como liveness leve, conforme a spec 002.

## Contexto Técnico

**Linguagem/Versão**: Go (módulo `go 1.26.1`; baseline com Go 1.27.1)

**Dependências principais**: `core-sdk/pkg/otel/tracer`, `core-sdk/pkg/zap/logger`, `core-sdk/pkg/otel` (`Start`/`Shutdown`), `go.opentelemetry.io/otel`, Echo v4

**Armazenamento**: nenhum nesta camada

**Testes**: sem testes de span/log; a verificação é por inspeção de código e comparação com os testes de `Service` (spec 001)

**Mocks**: não aplicável nesta camada

**Plataforma alvo**: container Kubernetes com `OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector-service:4317` e `OTEL_SERVICE_NAME=balances-api`

**Restrições**: nomes de span, configuração do middleware e sequência de shutdown congelados; spans permanecem no core até a spec 005

**Escopo neste repo**: 1 arquivo de inicialização, 4 spans, 1 middleware, 4 pontos de log em adapters, 2 env vars de coletor

## Verificação da Constituição

| Princípio | Situação | Evidência |
|---|---|---|
| I. Separação em camadas | **Desvio registrado** — spans dentro do domínio | `service.go`; spec 005 mantém no core nesta fase |
| II. O domínio valida, a interface transporta | Passa | middleware só transporta/loga; validação em `domain` |
| III. Configuração por ambiente, com singleton explícito | **Desvio registrado** — `config.New` na goroutine do servidor | `cmd/main.go`; spec 005 corrige |
| IV. Observabilidade é parte do request | Passa | `tracer.NewSpanFromContext`, `logger.InterceptorWithConfig`, OTLP no k8s |
| V. Consultas são desenhadas no modelo de dados | Não aplicável | nenhuma consulta nesta camada |
| VI. Testes unitários determinísticos | Passa com lacuna documentada | suíte de domínio roda sem coletor |
| VII. Infraestrutura como código | Passa | `scripts/k8s/app-deployment.yaml`, `go.mod` |

## Estrutura do Projeto

### Documentação (esta funcionalidade)

```text
docs/specs/004-observabilidade-e-shutdown/
├── spec.md
├── plan.md
└── tasks.md
```

### Código-fonte (raiz do repositório)

```text
cmd/main.go                                   # otel.Start, sinais, shutdown HTTP → otel
internal/domain/accounts/service.go           # spans Service::* e logger.Error
internal/infra/rest/server/server.go          # middleware, GET /, Shutdown
internal/infra/database/mysqldb/*.go          # logs estruturados de falha SQL
internal/infra/message/service.go             # log em Close
scripts/k8s/app-deployment.yaml               # OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_SERVICE_NAME
go.mod                                        # otel + core-sdk
```

**Decisão de estrutura**: manter a instrumentação dentro do caso de uso (`Service`), sem decorators. Isso preserva o comportamento atual e evita duplicar spans; a spec 005 registra o desvio e deixa decorators para fase opcional futura.

## Detalhamento do fluxo

```text
main()
 ├─ server.New()                     → Echo + rotas + middleware de log
 ├─ goroutine: config.New(...)       → s.Start(c.AppPort)
 ├─ otel.Start(context.Background())
 ├─ signal.Notify(SIGINT, SIGTERM)   → <-quit
 ├─ print "starting graceful shutdown..."
 ├─ ctx, cancel = WithTimeout(25s)
 ├─ s.Shutdown(ctx)                  → Echo.Shutdown
 ├─ otel.Shutdown(ctx)
 └─ print "graceful shutdown done"
```

## Complexidade Conhecida

| Desvio | Por que existe | Alternativa mais simples considerada |
|---|---|---|
| `otel.Start` depois do start do servidor | Ordem histórica em `main` | Iniciar antes; rejeitado para não alterar comportamento |
| `config.New(...)` dentro da goroutine | Configuração tardia | Mover para a raiz — previsto na spec 005 |
| `fmt.Println` junto de `logger` | Migração parcial para log estruturado | Unificar; fora do escopo desta spec |
| Sem `/health/ready` | Só liveness implementada | `HealthChecker` injetado — proposta da spec 005 |
| Sem correlation ID em resposta | Middleware do core-sdk não injeta header | Gerar `x-cid`; exigiria mudança de contrato HTTP |
| Spans no domínio | Implementação atual | Decorators nas bordas; diferidos pela spec 005 |
| Sem `Close()` de MySQL/SNS no `main` | Handlers fecham por request | Compartilhar adapters e fechar na raiz — spec 005 |

## Evolução Observada (histórica)

1. `a63fc2e`, `47fa6fe` — log personalizado e middleware de requisição.
2. `bb5f9db` — estrutura de spans nos casos de uso.
3. `eab8468`, `8fb2ccc` — configuração e variáveis de ambiente/portas.
4. `ecbf7c2` — refatoração incluindo observabilidade (core-sdk, OTLP).
5. `27ea5b2`, `16ba535` — ajustes finais em spans e logs.

## Verificação

```bash
# executar na raiz do repo
go build ./...
go vet ./...
grep -R 'tracer.NewSpanFromContext\|SetError\|AddAttributes' internal/domain/accounts/service.go
grep -R 'InterceptorWithConfig\|RedactFields\|MaxBodySize' internal/infra/rest/server/server.go
grep -R 'otel.Start\|otel.Shutdown\|WithTimeout\|signal.Notify' cmd/main.go
```

## Dependências e Ordem de Execução

- Depende da spec 001 (métodos de `Service` instrumentados) e da spec 002 (`Server`, middleware e `GET /`).
- A spec 003 fornece os adapters que registram logs de SQL/SNS.
- A spec 005 pode mover arquivos e injetar dependências, mas não altera os itens medidos por SC-001 a SC-007.
