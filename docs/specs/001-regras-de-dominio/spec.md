# Especificação da Funcionalidade: Regras de Domínio de Contas, Saldos e Lançamentos — `balances-api`

**Branch da funcionalidade**: `001-regras-de-dominio`

**Criado**: 2026-11-22

**Repo**: `github.com/clodoaldomarques/balances-api`

**Status**: Implementada (derivada do código em `16ba535`) — especificação **retroativa e descritiva**, reconstruída por engenharia reversa de `main`

**Entrada**: `internal/domain/accounts/**`, `account_test.go`, `service_test.go`, `docs/model/03 - C4 - Domain.puml` e histórico Git (commits `f7bbfa9`, `9dc1ade`, `1dc7715`, `16e88c2`, `c32ce93`, `4c2e592`, `ae352f8`).

> **Escopo**: rastreabilidade. Esta spec documenta o comportamento **já existente**; não autoriza mudança de regra, nome, mensagem de erro ou contrato. Onde o código apresenta desvios ou erros de digitação, eles são registrados como **comportamento atual congelado** — correções devem virar spec própria. Os testes citados são os que hoje estão no repositório.

## Contexto e Motivação

O pacote `internal/domain/accounts` concentra o modelo de contas e a lógica de negócio: `account.go` (agregado e mutações), `balances.go` (saldo disponível com limite), `limits.go` (consistência de limites), `operations.go` (débito/crédito), `entry.go`/`impact.go` (lançamento e impactos), `errors.go` (erros de domínio), `events.go` (contratos de evento) e `service.go` (orquestração do caso de uso). `interfaces.go` declara `Repository`, `Topic` e `Event`, com mocks gerados em `mock.go`.

As regras são puras e determinísticas: não consultam banco, não fazem I/O e usam `shopspring/decimal` para valores monetários. A suíte `account_test.go`/`service_test.go` cobre as transições de status, limites, operações e falhas. Esta spec existe para tornar esse conjunto rastreável: que regra existe, onde está implementada, qual teste a protege e em que commit ela apareceu.

## Cenários de Usuário e Testes *(obrigatório)*

### US1 — Representar a conta, seus saldos e seus limites (Prioridade: P1)

Como sistema de pagamentos, quero que uma conta seja representada por um agregado com saldos, limites, status e versão, para que cada alteração seja auditável.

**Por que esta prioridade**: é o contrato de dados usado por todas as camadas (HTTP, MySQL, eventos e testes); qualquer divergência de tipo ou chave altera o comportamento externo.

**Teste independente**: `go test ./internal/domain/accounts/... -run 'TestAccount'` com o helper `buildAccount()` (`account_test.go`).

**Cenários de aceitação**:

1. **Dado** `account.go`, **Quando** uma conta é criada, **Então** ela contém `AccountID int64`, `OrgID string`, mapas `Limits`/`Balances` de `decimal.Decimal`, `CreatedAt`, `UpdatedAt`, `Status` e `Version`.
2. **Dados** os status `active`, `only_credit`, `only_debit` e `inactive` (`Inative` no código), **Quando** comparados como `string(Status)`, **Então** produzem exatamente esses quatro valores.
3. **Dadas** as chaves de saldo `available`, `savings`, `blocked` e de limite `max_limit`, `total_limit`, `overdraft_limit`, **Quando** usadas nos mapas, **Então** são constantes exportadas (`Available`, `Savings`, `Blocked`, `MaxLimit`, `TotalLimit`, `OverdraftLimit`).
4. **Dada** qualquer mutação bem-sucedida (`ChangeStatus`, `ChangeLimit`, `ChangeBalances`), **Quando** ela termina, **Então** `IncreaseVersion()` incrementa `Version` e atualiza `UpdatedAt` com `time.Now()` exatamente uma vez por operação.

---

### US2 — Aplicar regras de débito e crédito com validação de saldo (Prioridade: P1)

Como sistema de pagamentos, quero processar impactos com segurança, para que débito não ultrapasse saldo/limite e crédito respeite o status da conta.

**Por que esta prioridade**: são as regras financeiras centrais do serviço e as mais sensíveis a regressão.

**Teste independente**: `TestAccount_ChangeBalances` (`account_test.go`) e o caso "insuficient balance" de `TestService_ProcessEntry` (`service_test.go`).

**Cenários de aceitação**:

1. **Dado** um `Impact` com `Operation` `CREDIT` ou `DEBIT`, **Quando** `ChangeBalances` é chamado, **Então** a operação é resolvida pelo mapa `Operations` e o saldo correspondente é somado ou subtraído.
2. **Dada** uma conta `active` ou `only_credit`, **Quando** recebe crédito, **Então** o valor é creditado; `only_debit` e `inactive` retornam `ErrValidateOperation`.
3. **Dada** uma conta `active` ou `only_debit`, **Quando** recebe débito, **Então** o valor é debitado; `only_credit` e `inactive` retornam `ErrValidateOperation`.
4. **Dada** a regra `consider_available_balance`, **Quando** o débito é avaliado, **Então** é considerado `Balances[available] + Limits[total_limit]`; saldo insuficiente retorna `ErrInsuficientBalance`.
5. **Dadas** as regras `consider_available_savings` e `consider_blocked_balance`, **Quando** o débito é avaliado, **Então** o saldo de `savings`/`blocked` é comparado ao valor; saldo insuficiente retorna `ErrInsuficientBalance`.
6. **Dado** um impacto de crédito com `Rules` não vazio, **Quando** `ChangeBalances` é chamado, **Então** retorna `ErrValidateOperation` sem alterar saldos.
7. **Dada** uma sequência de impactos, **Quando** um deles falha, **Então** a validação para e nenhuma alteração de versão é persistida pelo agregado nessa chamada.

---

### US3 — Alterar limites e status da conta com consistência (Prioridade: P1)

Como sistema de contas, quero atualizar limites e status respeitando as regras de consistência, para evitar estados inválidos.

**Por que esta prioridade**: os limites participam diretamente da validação de saldo da US2.

**Teste independente**: `TestAccount_ChangeLimit` e `TestAccount_ChangeStatus` (`account_test.go`).

**Cenários de aceitação**:

1. **Dado** `ChangeLimit(max_limit, valor)`, **Quando** o novo valor for menor que `total_limit`, **Então** retorna `ErrValidateLimit` com a mensagem `new limit can not less than total limit`.
2. **Dado** `ChangeLimit(total_limit, valor)`, **Quando** o novo valor for maior que `max_limit` ou `valor + available` for negativo, **Então** retorna `ErrValidateLimit` (`new limit can not great than max limit` / `new limit can not less than available balance`).
3. **Dado** `ChangeLimit(overdraft_limit, valor)`, **Quando** o valor é informado, **Então** a validação atual não impõe restrição (`validateChangeOverdraftLimit` retorna `nil`) e o limite é aplicado.
4. **Dada** uma conta `inactive`, **Quando** `ChangeLimit` é chamado para qualquer limite, **Então** retorna `ErrAccountDisabled` (`this Account is disabled for this operation`).
5. **Dado** `ChangeStatus`, **Quando** recebe qualquer `Status` válido, **Então** o valor é atribuído diretamente e a versão é incrementada — não há validação de transição de estado.

---

### US4 — Orquestrar os casos de uso da conta (Prioridade: P1)

Como chamador da API, quero um único conjunto de operações para criar conta, alterar limites/status e processar entradas, com persistência antes da publicação do evento.

**Por que esta prioridade**: `Service` é a fronteira entre regra de negócio e infraestrutura; hoje ele vive no pacote de domínio e é referência direta da spec 005.

**Teste independente**: `go test ./internal/domain/accounts/... -run 'TestService|TestNewService'`, usando `NewMockRepository`/`NewMockTopic` de `mock.go`.

**Cenários de aceitação**:

1. **Dado** `NewService(repository, topic)`, **Quando** construído, **Então** guarda as duas dependências em `rep` e `pub`.
2. **Dada** a chamada `CreateNewAccount`, **Quando** `SaveNewAccount` falha, **Então** o erro é devolvido, `Emit` não é chamado e nenhum evento é publicado.
3. **Dada** a chamada `CreateNewAccount`, **Quando** a persistência succeede, **Então** `buildCreateAccountEvent` gera `CreateAccountEvent` e `pub.Emit` é chamado uma vez.
4. **Dadas** `UpdateAccountLimits` e `UpdateAccountStatus`, **Quando** a conta não é encontrada ou a atualização falha, **Então** o erro é propagado sem emissão; em caso de sucesso, `UpdateAccountEvent` é emitido.
5. **Dada** a chamada `ProcessEntry`, **Quando** a recuperação, a validação de saldo ou `SaveEntryAndUpdateAccount` falham, **Então** o erro é devolvido e o span recebe `SetError`; em caso de sucesso, `ProcessEntryEvent` é emitido.
6. **Dado** o retorno de qualquer caso de uso, **Quando** a operação falha, **Então** uma `Account{}` vazia é devolvida junto com o erro.

---

### US5 — Defender os contratos de evento de domínio (Prioridade: P2)

Como consumidor assíncrono (worker), quero tipos e payloads de evento estáveis, para que o contrato SNS/SQS não mude sem versão nova.

**Por que esta prioridade**: `Event` é uma interface de domínio (`EventType()`, `EventData()`) e o payload é serializado em JSON no publicador.

**Teste independente**: inspeção de `events.go` e comparação com o payload publicado por `internal/infra/message/service.go`.

**Cenários de aceitação**:

1. **Dados** os tipos `create_account`, `update_account` e `process_entry`, **Quando** `EventType()` é chamado, **Então** retorna exatamente esses valores.
2. **Dado** `CreateAccountEvent`, **Quando** serializado, **Então** contém `account_id`, `org_id`, `limits`, `balances`, `created_at`, `status`, `version`.
3. **Dado** `UpdateAccountEvent`, **Quando** serializado, **Então** contém `account_id`, `org_id`, `limits`, `balances`, `status`, `updated_at`, `version`.
4. **Dado** `ProcessEntryEvent`, **Quando** serializado, **Então** contém `account_id`, `org_id`, `tracking_id`, `impacts`, `limits`, `balances`, `version`, `created_at`; cada `ImpactEvent` contém `balance`, `operation`, `amount` e `rules` (omitido quando vazio).
5. **Dado** `EventData()`, **Quando** chamado em qualquer evento, **Então** devolve o próprio valor concreto, sem conversão.

---

### Casos Limite

- `errors.go` usa `ErrInsuficientBalance` (com "Insuficient"), `ErrAccountDisabled`, `ErrValidateLimit` e `ErrValidateOperation`; mensagens e nomes são contrato congelado.
- A constante `Inative` serializa como `"inactive"`; o nome do identificador é histórico e não deve ser "corrigido" junto de outras specs.
- `validateChangeOverdraftLimit` aceita qualquer valor; não há regra de overdraft implementada neste repo.
- `creditFun`/`debitFun` assumem que a chave de saldo existe no mapa; mapas ausentes produzem zero-value de `decimal.Decimal`.
- `Operations`, `rules` e `limits` são mapas de pacote, não injetados; adicionar chave é mudança de comportamento.
- `ChangeBalances` incrementa a versão uma única vez após todos os impactos; não há versionamento por impacto.
- `Service` cria spans e logs (`core-sdk`), portanto `internal/domain/accounts` depende de observabilidade — desvio registrado e preservado até a spec 005.
- `pub.Emit` retorna `error`, mas `Service` descarta o retorno; comportamento atual congelado.
- `daily_balances` existe no SQL, porém nenhuma regra ou caso de uso deste repo a grava — o cálculo pertence ao `balances-worker`.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: A entidade `Account` DEVE manter os campos e tipos atuais, com saldos/limites em `map[string]decimal.Decimal`.
- **FR-002**: Os status DEVEM ser `active`, `only_credit`, `only_debit` e `inactive`, representados pelas constantes `Active`, `OnlyCredit`, `OnlyDebit` e `Inative`.
- **FR-003**: As chaves de saldo DEVEM ser `available`, `savings` e `blocked`, e as de limite `max_limit`, `total_limit` e `overdraft_limit`.
- **FR-004**: Toda mutação bem-sucedida DEVE incrementar `Version` e atualizar `UpdatedAt` por meio de `IncreaseVersion()`.
- **FR-005**: `ChangeBalances` DEVE aplicar impactos na ordem recebida e validar status, regras e operação antes de alterar saldos.
- **FR-006**: O débito em `available` DEVE considerar o limite `total_limit` na verificação de saldo.
- **FR-007**: Crédito com `Rules` NÃO DEVE ser aceito; débito com regras DEVE validar cada regra.
- **FR-008**: Limites `max_limit` e `total_limit` DEVEM respeitar as relações atuais; `overdraft_limit` NÃO DEVE receber validação adicional.
- **FR-009**: `ChangeStatus` NÃO DEVE validar transição de estado.
- **FR-010**: `Service` DEVE expor `CreateNewAccount`, `UpdateAccountLimits`, `UpdateAccountStatus` e `ProcessEntry`, com persistência anterior à emissão de evento.
- **FR-011**: Erros de repositório ou de regra DEVEM impedir a emissão do evento correspondente.
- **FR-012**: `Repository`, `Topic` e `Event` DEVEM permanecer como contratos do domínio, com mocks gerados por `go:generate`.
- **FR-013**: Os eventos `create_account`, `update_account` e `process_entry` e seus campos JSON NÃO PODEM mudar nesta spec.
- **FR-014**: As mensagens de erro atuais DEVEM ser preservadas, incluindo grafias e erros de digitação.

### Entidades Principais

- **Account**: agregado de saldos, limites, status e versão.
- **Entry / Impact**: lançamento e seus impactos por saldo, operação, valor e regras.
- **Service**: caso de uso que coordena `Repository` e `Topic`.
- **Repository / Topic / Event**: portas historicamente declaradas em `interfaces.go`.
- **ErrInsuficientBalance / ErrAccountDisabled / ErrValidateLimit / ErrValidateOperation**: erros de domínio.
- **CreateAccountEvent / UpdateAccountEvent / ProcessEntryEvent / ImpactEvent**: contratos de domínio publicados como eventos.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: `go test ./internal/domain/accounts/...` passa sem alteração de código (baseline observado: OK).
- **SC-002**: `account_test.go` cobre status, limites, débito, crédito, transferência entre saldos e todas as mensagens de erro listadas.
- **SC-003**: `service_test.go` cobre os quatro casos de uso, incluindo falha de repositório antes de `Emit`.
- **SC-004**: Nenhuma regra, constante ou mensagem de erro desta spec é alterada; `git diff` de `internal/domain/accounts/**` é vazio.
- **SC-005**: Os três `EventType()` e os campos JSON dos eventos permanecem idênticos aos de `events.go`.
- **SC-006**: O diagrama `docs/model/03 - C4 - Domain.puml` permanece compatível com as entidades `Account`, `Entries` e `Impact`.

## Evidência de Implementação (engenharia reversa)

| Comportamento | Arquivo(s) | Histórico / teste |
|---|---|---|
| Modelo e mutações da conta | `internal/domain/accounts/account.go` | `f7bbfa9`, `9dc1ade`; `account_test.go` |
| Regras de saldo e limite | `internal/domain/accounts/balances.go`, `limits.go` | `16e88c2`, `c32ce93`; `TestAccount_ChangeBalances`, `TestAccount_ChangeLimit` |
| Débito/crédito e status | `internal/domain/accounts/operations.go` | `9dc1ade`, `1dc7715`; `TestAccount_ChangeBalances` |
| Casos de uso e portas | `internal/domain/accounts/service.go`, `interfaces.go`, `mock.go` | `dd57b8c`, `d8b5ad3`; `service_test.go` |
| Contratos de evento | `internal/domain/accounts/events.go` | `4c2e592`, `d8b5ad3`; `service_test.go` |
| Mensagens de erro padronizadas | `internal/domain/accounts/errors.go` | `ae352f8`; asserções de `e.Error()` nos testes |

## Suposições

- O estado atual do código é a fonte da verdade; esta spec não corrige nomes, grafias ou regras.
- `go test`, `go build` e `go vet` foram executados em 2026-11-22 sobre `go.mod/go.sum` presentes no diretório de trabalho.
- A suíte existente é suficiente como baseline; testes adicionais de contrato são responsabilidade da spec 005.
- A separação futura entre `domain`, `application` e `ports` está descrita na spec 005 e não faz parte desta.
