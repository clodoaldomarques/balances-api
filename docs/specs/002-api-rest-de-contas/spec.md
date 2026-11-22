# Especificação da Funcionalidade: API REST de Contas e Lançamentos — `balances-api`

**Branch da funcionalidade**: `002-api-rest-de-contas`

**Criado**: 2026-11-22

**Repo**: `github.com/clodoaldomarques/balances-api`

**Status**: Implementada (derivada do código em `16ba535`) — especificação **retroativa e descritiva**, reconstruída por engenharia reversa de `main`

**Entrada**: `internal/infra/rest/**`, `cmd/main.go`, `docs/model/01 - C4 - Context.puml`, `docs/model/02 - C4 - Container.puml` e histórico Git (`b0d7f29`, `294b83c`, `c53c5df`, `13c0ff2`, `8fb2ccc`, `dd57b8c`).

> **Nota de migração (spec 005)**: os caminhos `internal/domain/accounts/**` e `internal/infra/**` abaixo são históricos (commit `16ba535`). Após a spec 005, o código equivalente está em `internal/core/accounts/**` e `internal/adapters/**`; as regras e os contratos descritos aqui permanecem os mesmos. Veja o índice em [`README.md`](../README.md) e a convenção em [`../constitution.md`](../constitution.md).

> **Escopo**: documenta o contrato HTTP **atual**, inclusive rotas, códigos de status, códigos de erro e mensagens. Não há collection Postman versionada neste repo; a verificação usa as rotas implementadas e os testes existentes. Nenhuma mudança de comportamento é autorizada — a spec 005 congela este contrato.

## Contexto e Motivação

O servidor Echo é construído em `internal/infra/rest/server/server.go`, que registra o validador customizado, o middleware de log, a rota de health e as quatro rotas de contas. Os handlers em `internal/infra/rest/accounts/handler.go` fazem bind e validação dos DTOs definidos em `types.go`, convertem para entidades de domínio e chamam `accounts.Service`. As respostas de sucesso usam DTOs próprios e as falhas usam `ErrorResponse{code, error}` com `ERR0001`, `ERR0002` ou `ERR0003`, sempre com status `400`.

Essa superfície é o único contrato síncrono do serviço: é por ela que o "Sistema de Pagamentos" solicita débitos/créditos e que o "Sistema de Contas" cria limites (ver C4 de contexto/container). Esta spec torna cada rota, DTO e erro rastreável.

## Cenários de Usuário e Testes *(obrigatório)*

### US1 — Criar uma conta com saldos e limites iniciais (Prioridade: P1)

Como Sistema de Contas, quero enviar os saldos e limites iniciais, para que uma nova conta seja aberta com versão 1.

**Por que esta prioridade**: é a porta de entrada para todo o ciclo de vida da conta e a origem do evento `create_account`.

**Teste independente**: `POST /accounts` com payload válido contra LocalStack/MySQL, verificando `201` e o corpo com `status: "active"` e `version: 1`.

**Cenários de aceitação**:

1. **Dado** `POST /accounts`, **Quando** o corpo contém `account_id`, `org_id`, `limits` e `balances` válidos, **Então** o handler responde `201` com `PostAccountResponse`.
2. **Dado** `PostAccountRequest.ToEntity()`, **Quando** convertido, **Então** preenche `CreatedAt`/`UpdatedAt` com `time.Now()`, `Status = Active` e `Version = 1`.
3. **Dado** um corpo inválido para bind, **Quando** o handler processa, **Então** responde `400` com `code: "ERR0001"`.
4. **Dado** um corpo que falha nas validações de domínio do DTO, **Quando** processado, **Então** responde `400` com `code: "ERR0002"`.
5. **Dada** uma falha de `CreateNewAccount`, **Quando** o erro chega ao handler, **Então** responde `400` com `code: "ERR0003"`.

---

### US2 — Atualizar os limites de uma conta (Prioridade: P1)

Como Sistema de Contas, quero alterar limites informando `orgID` e `accountID`, para que novas regras de saldo passem a valer.

**Por que esta prioridade**: a rota é usada para ajustar `max_limit`, `total_limit` e `overdraft_limit` sem recriar a conta.

**Teste independente**: `PUT /accounts/TN-12345678/23052013/limits` com `{"limits":{"max_limit":150}}`, verificando `200` e versão incrementada.

**Cenários de aceitação**:

1. **Dado** `PUT /accounts/:orgID/:accountID/limits`, **Quando** `accountID` não é inteiro `int64`, **Então** responde `400` com `ERR0001`.
2. **Dados** parâmetros válidos e corpo com `limits` não vazio, **Quando** a regra de domínio aceita, **Então** responde `200` com `PutAccountResponse`.
3. **Dado** um limite desconhecido ou negativo, **Quando** validado, **Então** responde `400` com `ERR0002` e mensagem `invalid limit: <chave>` ou `limit <chave> can not less then zero`.
4. **Dada** uma falha de regra ou persistência, **Quando** `UpdateAccountLimits` retorna erro, **Então** responde `400` com `ERR0003`.

---

### US3 — Alterar o status de uma conta (Prioridade: P1)

Como Sistema de Contas, quero habilitar/desabilitar operações via status, para controlar crédito e débito permitidos.

**Por que esta prioridade**: o status participa diretamente das regras `creditFun`/`debitFun` da spec 001.

**Teste independente**: `PUT /accounts/:orgID/:accountID/status` com `{"status":"only_credit"}`, verificando `200` e `status: "only_credit"` na resposta.

**Cenários de aceitação**:

1. **Dado** `status` entre `active`, `inactive`, `only_credit` e `only_debit`, **Quando** a rota é chamada, **Então** o valor é convertido para `accounts.Status` e enviado ao caso de uso.
2. **Dado** qualquer outro valor, **Quando** validado, **Então** responde `400` com `ERR0002` e mensagem `invalid status: <valor>`.
3. **Dada** uma conta inexistente, **Quando** o caso de uso retorna erro, **Então** responde `400` com `ERR0003` — não há mapeamento para `404` nesta versão.
4. **Dado** sucesso, **Quando** o handler monta a resposta, **Então** usa `AccountToPutAccountResponse`.

---

### US4 — Processar um lançamento (Prioridade: P1)

Como Sistema de Pagamentos, quero enviar um `Entry` com seus impactos, para que saldos sejam debitados/creditados e um evento seja publicado.

**Por que esta prioridade**: é a operação financeira da API e a origem do evento `process_entry`.

**Teste independente**: `POST /accounts/entries` com um impacto de débito válido, verificando `200`, saldo atualizado e evento publicado.

**Cenários de aceitação**:

1. **Dado** `POST /accounts/entries`, **Quando** `tracking_id`, `account_id`, `org_id` e `impacts` estão presentes, **Então** o DTO `PostEntryRequest` é convertido por `ToEntity()` com `CreatedAt = time.Now()`.
2. **Dado** cada `PostImpactRequest`, **Quando** validado, **Então** `balance` deve estar entre `available`, `savings`, `blocked`, `operation` entre `CREDIT`/`DEBIT` e `rules` entre as três regras `consider_*`.
3. **Dada** uma regra inválida em `rules`, **Quando** validada, **Então** responde `400` com `ERR0002` e mensagem `invalid consider: <regra>`.
4. **Dado** saldo insuficiente, **Quando** o domínio retorna `ErrInsuficientBalance`, **Então** responde `400` com `ERR0003` e a mensagem do erro de domínio.
5. **Dado** sucesso, **Quando** o handler responde, **Então** usa `AccountToPostAccountResponse` com status `200`.

---

### US5 — Validar requisições e padronizar respostas de erro (Prioridade: P1)

Como cliente da API, quero receber códigos de erro previsíveis, para distinguir falha de formato, de validação e de negócio.

**Por que esta prioridade**: todos os handlers compartilham o mesmo padrão; mudá-lo quebra consumidores.

**Teste independente**: comparação das três saídas de erro em uma rota, verificando corpo `{"code":"...","error":"..."}` e status `400`.

**Cenários de aceitação**:

1. **Dado** `CustomValidator.Validate`, **Quando** chamado, **Então** executa `validator.Struct(i)` e, se o objeto implementar `EntityRequest`, executa também `Validate()`.
2. **Dado** `ERR0001`, **Quando** aparece, **Então** indica falha de bind ou de parse de `accountID`.
3. **Dado** `ERR0002`, **Quando** aparece, **Então** indica falha do validador customizado.
4. **Dado** `ERR0003`, **Quando** aparece, **Então** indica erro devolvido pelo caso de uso.
5. **Dado** qualquer erro nesta versão, **Quando** serializado, **Então** o status HTTP é `400` e o corpo contém somente `code` e `error`.

---

### US6 — Responder ao health check HTTP (Prioridade: P2)

Como orquestrador Kubernetes, quero uma rota leve de liveness, para consultar se o processo está no ar.

**Por que esta prioridade**: é a única rota de health existente neste repo; não há `/health/ready` com verificação de dependências.

**Teste independente**: `GET /` retorna `200` com `{"data":"Server is up and running"}`.

**Cenários de aceitação**:

1. **Dado** o servidor iniciado, **Quando** `GET /` é chamada, **Então** retorna `200` com o corpo exato `{"data":"Server is up and running"}`.
2. **Dada** a rota `/`, **Quando** comparada às demais, **Então** não consulta MySQL, SNS ou SQS.

---

### Casos Limite

- A ordem de registro das rotas em `routes()` é: middleware de log, `GET /`, `POST /accounts`, `PUT .../limits`, `PUT .../status`, `POST /accounts/entries`.
- Valores monetários são transportados como `decimal.Decimal` (`shopspring/decimal`), serializados como texto decimal — nunca como float.
- `PostImpactRequest.Amount` não possui validação de valor mínimo; valor negativo depende da regra de domínio.
- `PostAccountRequest.Validate()` exige mapas `limits` e `balances` não vazios; `PutAccountLimitsRequest.Validate()` exige apenas `limits`.
- `ErrorResponse` não expõe detalhes de stack; mensagens vêm de `err.Error()`.
- Os handlers constroem `mysqldb.NewRepository` e `message.NewTopic` a cada requisição e os fecham com `defer` — comportamento atual que a spec 005 propõe mudar.
- O nome dos campos de resposta usa `omitempty`; campos zero-value podem desaparecer do JSON.
- A rota `/accounts/entries` é registrada depois de `/accounts/:orgID/:accountID/...`, mas usa path literal distinto; o comportamento atual deve ser preservado.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O servidor DEVE registrar exatamente as rotas `GET /`, `POST /accounts`, `PUT /accounts/:orgID/:accountID/limits`, `PUT /accounts/:orgID/:accountID/status` e `POST /accounts/entries`.
- **FR-002**: `POST /accounts` DEVE responder `201` em sucesso e devolver `PostAccountResponse`.
- **FR-003**: `PUT .../limits` e `PUT .../status` DEVEM responder `200` em sucesso e devolver `PutAccountResponse`.
- **FR-004**: `POST /accounts/entries` DEVE responder `200` em sucesso e devolver `PostAccountResponse`.
- **FR-005**: `GET /` DEVE responder `200` com `{"data":"Server is up and running"}`.
- **FR-006**: DTOs de requisição DEVEM implementar `Validate()` e ser validados pelo `CustomValidator` instalado no Echo.
- **FR-007**: Chaves de limite, saldo, operação, status e regra DEVEM ser validadas contra as listas atuais de `types.go`.
- **FR-008**: A conversão de DTO para entidade DEVE preencher `CreatedAt`/`UpdatedAt`, `Status` e `Version` conforme o comportamento atual.
- **FR-009**: Erros de bind/parse DEVEM usar `ERR0001`; erros de validação, `ERR0002`; erros de caso de uso, `ERR0003`.
- **FR-010**: Toda resposta de erro DEVE ter status `400` e corpo com `code` e `error`.
- **FR-011**: Os handlers NÃO DEVEM alterar regras de negócio; toda decisão financeira permanece no domínio (spec 001).
- **FR-012**: Valores monetários DEVEM ser transportados via `decimal.Decimal`, sem conversão para `float64`.
- **FR-013**: O contrato de rotas, status e JSON NÃO PODE mudar em decorrência da spec 005.
- **FR-014**: A ausência de `404`, de autenticação e de collection Postman versionada é parte do estado atual documentado.

### Entidades Principais

- **Server / routes()**: composição das rotas no Echo.
- **CustomValidator**: ponte entre `validator` e `EntityRequest.Validate()`.
- **PostAccountRequest / PostAccountResponse**: contrato de criação de conta.
- **PutAccountLimitsRequest / PutAccountStatusRequest / PutAccountResponse**: contrato de atualização.
- **PostEntryRequest / PostImpactRequest**: contrato de lançamento.
- **ErrorResponse**: envelope de erro (`code`, `error`).

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: As cinco rotas acima respondem exatamente com os status `201`, `200` e `400` documentados.
- **SC-002**: Nenhuma rota adicional ou renomeada existe em `server.go`.
- **SC-003**: Os três códigos `ERR0001`/`ERR0002`/`ERR0003` são os únicos emitidos pelos handlers.
- **SC-004**: O corpo do health check é idêntico ao de `HealthCheck` em `server.go`.
- **SC-005**: DTOs com chaves inválidas são rejeitados antes de chamar o domínio.
- **SC-006**: `go build ./...` e `go vet ./...` passam sem alteração de `internal/infra/rest/**`.
- **SC-007**: A spec 005 pode mover os handlers de pasta sem alterar um único path, status ou campo JSON desta spec.

## Evidência de Implementação (engenharia reversa)

| Comportamento | Arquivo(s) | Histórico |
|---|---|---|
| Rotas, validador, health | `internal/infra/rest/server/server.go` | `b0d7f29`, `dd57b8c` |
| Handlers e mapeamento de erro | `internal/infra/rest/accounts/handler.go` | `b0d7f29`, `c53c5df`, `ae352f8` |
| DTOs e validações | `internal/infra/rest/accounts/types.go` | `294b83c`, `bb2a132`, `8fb2ccc` |
| Porta HTTP e inicialização | `cmd/main.go`, `scripts/k8s/app-service.yaml` | `13c0ff2`, `8fb2ccc` |
| Contrato de contexto/container | `docs/model/01` e `02` (`*.puml`) | `ee8b282` |

## Suposições

- O estado atual do código é a fonte da verdade; esta spec não corrige a ausência de `404` nem a duplicação de construção de adapters.
- Não há Collection Postman versionada; os cenários foram validados por leitura dos handlers e DTOs.
- Serialização de `decimal.Decimal` segue o comportamento padrão da biblioteca (`shopspring/decimal`).
- A spec 004 documenta o middleware de log e o shutdown; a spec 003 documenta porta e infraestrutura.
