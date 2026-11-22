# Plano de Implementação: API REST de Contas e Lançamentos — `balances-api`

**Branch**: `002-api-rest-de-contas` | **Data**: 2026-11-22 | **Spec**: [spec.md](spec.md)

**Repo**: `github.com/clodoaldomarques/balances-api`

**Entrada**: [spec.md](spec.md) deste diretório, `internal/infra/rest/**` e contrato C4 em `docs/model/01` e `02`.

**Nota**: plano **retroativo**. Descreve as decisões que produziram o código atual; `[x]` em [tasks.md](tasks.md) significa "implementado e verificável", não "tarefa a fazer".

## Resumo

Expor quatro operações de contas e um health check sobre Echo, com DTOs próprios para cada operação, validação em duas camadas (`validator` + `EntityRequest.Validate()`), conversão explícita para entidades de domínio e envelope de erro único (`ERR0001`/`ERR0002`/`ERR0003`, sempre HTTP 400). O servidor é criado uma vez em `cmd/main.go` e escuta na porta configurada.

## Contexto Técnico

**Linguagem/Versão**: Go (módulo `go 1.26.1`; baseline com Go 1.27.1)

**Dependências principais**: Echo v4, `go-playground/validator` (via `CustomValidator`), `core-sdk/pkg/zap/logger` (middleware), `shopspring/decimal`, domínio `internal/domain/accounts`

**Armazenamento**: nenhum diretamente; os handlers chamam `Service`, que usa as portas da spec 003

**Testes**: não há testes de handler neste repo; a verificação é feita por inspeção das rotas e por `go build`/`go vet` (cobertura de handler é incumbência da spec 005)

**Mocks**: não aplicável nesta camada no estado atual

**Plataforma alvo**: container Kubernetes, Service do tipo `LoadBalancer` na porta 5002

**Restrições**: paths, status codes, payloads e códigos de erro congelados; sem autenticação

**Escopo neste repo**: 1 servidor, 5 rotas, 7 DTOs de requisição/resposta, 1 envelope de erro, 1 validador customizado

## Verificação da Constituição

| Princípio | Situação no código | Evidência |
|---|---|---|
| I. Separação em camadas | Passa com ressalva — handlers dependem diretamente de adapters MySQL/SNS | `handler.go` importa `mysqldb` e `message`; spec 005 corrige |
| II. O domínio valida, a interface transporta | Passa | validação de formato/chaves em `types.go`; regras em `domain` |
| III. Configuração por ambiente, com singleton explícito | Passa com ressalva — porta aplicada por opção em `main` | `config.New(config.WithAppPort(5002), ...)` |
| IV. Observabilidade é parte do request | Passa — middleware instalado em `routes()` | `logger.InterceptorWithConfig` |
| V. Consultas são desenhadas no modelo de dados | Não aplicável | nenhuma consulta nesta camada |
| VI. Testes unitários determinísticos | **Lacuna documentada** — sem teste de handler | spec 005 T019 prevê fake da porta de entrada |
| VII. Infraestrutura como código | Passa | `scripts/k8s/app-service.yaml`, Dockerfile |

## Estrutura do Projeto

```text
cmd/main.go                                  # cria o server e inicia na porta configurada
internal/infra/rest/server/server.go         # Echo, rotas, health, CustomValidator, middleware
internal/infra/rest/accounts/handler.go      # 4 handlers
internal/infra/rest/accounts/types.go        # DTOs, ToEntity, Validate, ErrorResponse
scripts/k8s/app-service.yaml                 # LoadBalancer 5002 -> targetPort 5002 -> nodePort 31002
scripts/docker/Dockerfile                    # EXPOSE 5000 (valor histórico)
docs/model/01 - C4 - Context.puml            # atores e sistema
docs/model/02 - C4 - Container.puml          # API, SNS, MySQL, consulta
```

**Decisão de contrato**: um DTO por operação, com `Validate()` próprio, em vez de um único modelo genérico. Isso mantém as regras de formato próximas do payload e permite que cada rota evolua sem afetar as demais. O lado negativo é a repetição de listas de chaves válidas (`validLimits`, `validBalances`, `validOperations`, `validStatus`, `validRules`), que permanece como estado atual.

## Mapeamento de Requisição → Resposta

| Rota | DTO de entrada | Caso de uso | DTO de saída | Sucesso | Erro |
|---|---|---|---|---|---|
| `POST /accounts` | `PostAccountRequest` | `CreateNewAccount` | `PostAccountResponse` | 201 | 400 `ERR000x` |
| `PUT /accounts/:orgID/:accountID/limits` | `PutAccountLimitsRequest` | `UpdateAccountLimits` | `PutAccountResponse` | 200 | 400 `ERR000x` |
| `PUT /accounts/:orgID/:accountID/status` | `PutAccountStatusRequest` | `UpdateAccountStatus` | `PutAccountResponse` | 200 | 400 `ERR000x` |
| `POST /accounts/entries` | `PostEntryRequest` | `ProcessEntry` | `PostAccountResponse` | 200 | 400 `ERR000x` |
| `GET /` | — | — | `{"data":"Server is up and running"}` | 200 | — |

## Complexidade Conhecida

| Desvio | Por que existe | Decisão (alternativa considerada) |
|---|---|---|
| Todos os erros são 400 | Implementação original dos handlers | Congelar; `404` caso a caso em spec futura |
| Handlers constroem adapters por request | Composição histórica | Documentar; spec 005 injeta portas |
| Sem teste de handler | Repo priorizou teste de domínio | Registrar lacuna; spec 005 adiciona fake |
| Porta 5000 (Docker/EXPOSE) × 5002 (opção/k8s Service) | Evolução da configuração | Documentar como divergência atual; não "corrigir" aqui |
| Health sem checagem de dependências | Somente liveness | Congelar; `/health/ready` é proposta da spec 005 |
| `Amount` sem validação de sinal | DTO não valida valor | Manter; domínio decide |

## Evolução Observada (histórica)

1. `b0d7f29` — primeiros handlers chamando o serviço.
2. `294b83c`, `bb2a132`, `c53c5df` — validação de entry e migração das validações para a camada de infra.
3. `13c0ff2`, `8fb2ccc` — porta do microserviço e leitura de variáveis de ambiente.
4. `ae352f8` — padronização das mensagens de erro.
5. `dd57b8c` — caminhos atuais (`internal/infra/rest/**`).

## Verificação

```bash
# executar na raiz do repo
go build ./...
go vet ./...
grep -R "s.http." internal/infra/rest/server/server.go      # confere as rotas
grep -R 'ERR000' internal/infra/rest/accounts/handler.go     # confere os códigos de erro
```

## Dependências e Ordem de Execução

- A spec 001 fornece as entidades e os erros exibidos em `ERR0003`.
- A spec 003 fornece a conexão MySQL e o publicador SNS usados pelos handlers.
- A spec 004 fornece o middleware de log instalado no servidor.
- A spec 005 move os handlers sem alterar nenhum item do mapa de requisição → resposta acima.
