# Plano de Implementação: Regras de Domínio de Contas, Saldos e Lançamentos — `balances-api`

**Branch**: `001-regras-de-dominio` | **Data**: 2026-11-22 | **Spec**: [spec.md](spec.md)

**Repo**: `github.com/clodoaldomarques/balances-api`

**Entrada**: [spec.md](spec.md) deste diretório, código de `internal/domain/accounts` e histórico Git citado na spec.

**Nota**: este plano é **retroativo**: descreve decisões já tomadas e a ordem histórica em que o domínio foi construído. `[x]` em [tasks.md](tasks.md) significa "implementado e verificável no código", não "tarefa a fazer". Nenhuma alteração de código é necessária para fechar esta spec.

## Resumo

Implementar o agregado `Account` com saldos e limites monetários, as regras de débito/crédito e validação de saldo, as transições de status/limite, o caso de uso `Service` e os contratos de evento publicados. Tudo em `internal/domain/accounts`, com valores `decimal.Decimal`, erros typed e testes unitários determinísticos com mocks gerados via `go:generate`.

## Contexto Técnico

**Linguagem/Versão**: Go (módulo declara `go 1.26.1` em `go.mod`; baseline executado com Go 1.27.1)

**Dependências principais**: `shopspring/decimal` (valores monetários), `core-sdk` (tracer/logger/commons.DecimalMap), `golang/mock` + `go:generate` (mocks), `testify` e `google/uuid` (testes)

**Armazenamento**: nenhum nesta camada — `Repository` é uma interface; a implementação MySQL está na spec 003

**Testes**: `account_test.go` (496 linhas, casos de status/limite/saldo) e `service_test.go` (389 linhas, casos de uso com mocks)

**Mocks**: `mock.go` gerado a partir de `interfaces.go` (`Repository`, `Topic`, `Event`)

**Plataforma alvo**: biblioteca de domínio, sem dependência de HTTP, MySQL ou AWS

**Restrições**: nenhuma alteração de regra, mensagem de erro ou contrato JSON; desvios atuais ficam congelados

**Escopo neste repo**: 8 arquivos de regra em `internal/domain/accounts`, 4 portas/contratos, 3 tipos de evento, 2 arquivos de teste

## Verificação da Constituição

| Princípio | Situação no código | Evidência |
|---|---|---|
| I. Separação em camadas | **Desvio registrado** — `Service` convive com entidades em `domain` | `internal/domain/accounts/service.go`; correção prevista na spec 005 |
| II. O domínio valida, a interface transporta | Passa | `ChangeLimit`, `ChangeBalances`, `account.go`; DTOs em `internal/infra/rest/accounts/types.go` |
| III. Configuração por ambiente, com singleton explícito | Não aplicável | nenhuma leitura de `config` no domínio |
| IV. Observabilidade é parte do request | **Desvio registrado** — spans/logger dentro de `Service` | `tracer.NewSpanFromContext` em `service.go` |
| V. Consultas são desenhadas no modelo de dados | Não aplicável | nenhuma consulta nesta camada |
| VI. Testes unitários determinísticos | Passa | `buildAccount()`, mocks de `Repository`/`Topic`, asserções de erro |
| VII. Infraestrutura como código | Não aplicável | ver spec 003 |

## Estrutura do Projeto

```text
internal/domain/accounts/
├── account.go         # Account, Status, ChangeStatus/ChangeLimit/ChangeBalances, IncreaseVersion
├── balances.go        # chaves de saldo e regras consider_*
├── limits.go          # chaves de limite e validações
├── operations.go      # CREDIT/DEBIT e mapa Operations
├── entry.go           # Entry
├── impact.go          # Impact
├── errors.go          # quatro tipos de erro de domínio
├── events.go          # Event, Create/Update/ProcessEntryEvent, ImpactEvent
├── interfaces.go      # Repository, Topic, Event + go:generate
├── mock.go            # gerado
├── account_test.go    # regras do agregado
└── service_test.go    # casos de uso com mocks
docs/model/03 - C4 - Domain.puml   # Account / Entries / Impact
```

**Decisão de modelo**: manter saldos e limites como mapas de `decimal.Decimal` em vez de campos fixos. Isso permite novas chaves sem migração de estrutura e é o formato gravado em JSON no MySQL e nos eventos. O custo é a ausência de validação em tempo de compilação para chaves — mitigada pelas listas de validação na camada HTTP (spec 002).

## Decisões de Regra

| Decisão | Comportamento escolhido | Racional / evidência |
|---|---|---|
| Valores monetários | `decimal.Decimal` | evita aritmética de ponto flutuante; usado em todo o repo |
| Versão otimista | `Version++` + `UpdatedAt` por mutação | gravada no MySQL e publicada nos eventos |
| Limite disponível | `available + total_limit` | `validateDebitAvailableBalance` em `balances.go` |
| Crédito com regras | rejeitado | `ChangeBalances` valida `len(i.Rules) > 0` para `CREDIT` |
| Status | sem validação de transição | `ChangeStatus` atribui diretamente |
| Overdraft | sem validação | `validateChangeOverdraftLimit` retorna `nil` |
| Erros | tipos concretos com `Error()` | `errors.go`; mensagens usadas em asserções e respostas HTTP |

## Complexidade Conhecida

| Desvio | Por que existe | Decisão (alternativa considerada) |
|---|---|---|
| Nomes com erro de grafia (`Inative`, `Insuficient`) | Histórico do código | Congelar; renomear quebra testes e contrato |
| `Service` no pacote de domínio | Estrutura original | Manter nesta spec; mover na 005 |
| Spans/logger no caso de uso | Refatoração de observabilidade de 2026 | Manter e documentar; decorators ficam para spec futura |
| `Emit` descarta erro | Implementação atual | Congelar; correção em spec própria |
| Mapas sem chave retornam zero | Comportamento de Go | Não criar validação nova no domínio |
| Regras duplicadas na DTO HTTP | Validação de transporte | Mantida; regra financeira continua no domínio |

## Evolução Observada (histórica)

1. **Fase A — modelo inicial**: `a70dfed` cria o pacote `account` com regras de domínio; `f7bbfa9`/`9a66326` recriam e organizam o domínio.
2. **Fase B — regras e testes**: `9dc1ade` implementa regras; `1dc7715` finaliza o domínio; `294b83c` inclui validação de entrada.
3. **Fase C — saldos e limites**: `16e88c2` ajusta regras com limite; `c32ce93` introduz saldo disponível/balanço diário; `8f9dc2d` ajusta o saldo.
4. **Fase D — eventos e erros**: `4c2e592` implementa eventos de domínio; `ae352f8` padroniza mensagens de erro.
5. **Fase E — consolidação**: `dd57b8c` desmembra a aplicação para os caminhos atuais; `d8b5ad3` remove o publisher próprio em favor do `core-sdk`.

Cada fase terminou com a suíte verde; o estado final é o baseline desta spec.

## Verificação

```bash
# executar na raiz do repo
go test ./internal/domain/accounts/...
go test ./internal/domain/accounts/... -run 'TestAccount_ChangeStatus|TestAccount_ChangeLimit|TestAccount_ChangeBalances'
go test ./internal/domain/accounts/... -run 'TestService|TestNewService'
go build ./...
go vet ./...
```

## Rastreabilidade

| Spec 001 | Spec 005 |
|---|---|
| Regras e entidades permanecem em `domain` | `internal/core/accounts/domain` sem alteração de regra |
| `Service` e `interfaces.go` no mesmo pacote | `Service` vai para `application`; portas para `ports` |
| `Event` como contrato de domínio | Permanece no core, não vira DTO de adaptador |
| Mensagens de erro congeladas | SC-005 exige nenhuma diferença observável |

## Dependências e Ordem de Execução

- A spec 001 é a base das specs 002 (consome as regras via DTOs), 003 (persiste e publica as entidades) e 004 (instrumenta o caso de uso).
- A spec 005 depende desta para garantir que o movimento de pacotes não altere regra alguma.
- Nenhuma tarefa pendente: a spec é fechada quando a evidência acima é verificada.
