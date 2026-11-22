# Especificação da Funcionalidade: Observabilidade, Health e Shutdown — `balances-api`

**Branch da funcionalidade**: `004-observabilidade-e-shutdown`

**Criado**: 2026-11-22

**Repo**: `github.com/clodoaldomarques/balances-api`

**Status**: Implementada (derivada do código em `16ba535`) — especificação **retroativa e descritiva**, reconstruída por engenharia reversa de `main`

**Entrada**: `cmd/main.go`, `internal/domain/accounts/service.go`, `internal/infra/rest/server/server.go`, `internal/infra/database/mysqldb/**`, `internal/infra/message/service.go`, `scripts/k8s/app-deployment.yaml`, `go.mod` e histórico Git (`a63fc2e`, `47fa6fe`, `bb5f9db`, `eab8468`, `ecbf7c2`, `27ea5b2`, `16ba535`).

> **Nota de migração (spec 005)**: os caminhos `internal/domain/accounts/**` e `internal/infra/**` abaixo são históricos (commit `16ba535`). Após a spec 005, o código equivalente está em `internal/core/accounts/**` e `internal/adapters/**`; as regras e os contratos descritos aqui permanecem os mesmos. Veja o índice em [`README.md`](../README.md) e a convenção em [`../constitution.md`](../constitution.md).

> **Escopo**: documenta a instrumentação e o ciclo de vida **atuais**. O contrato do endpoint de health (`GET /`) já está especificado na spec 002; aqui ele é tratado apenas como liveness do processo. Nenhuma mudança de comportamento é autorizada — a spec 005 preserva spans e logs dentro do core.

## Contexto e Motivação

A observabilidade do serviço tem três pontos: (1) spans abertos em cada método público de `Service` via `tracer.NewSpanFromContext`, com atributo `account_id`, `SetError` e, em falhas de `ProcessEntry`, o atributo `entry`; (2) middleware `logger.InterceptorWithConfig` no Echo, com limite de corpo, capture de request body e redação de campos sensíveis; (3) ciclo de vida em `cmd/main.go`, que inicia o OpenTelemetry, aguarda `SIGINT`/`SIGTERM` e encerra HTTP e otel com timeout de 25 segundos.

Os adapters também registram logs estruturados em falhas de conexão, prepare, retrieve e publicação. Esta spec torna rastreáveis nomes de span, atributos, configuração do interceptador, env vars do coletor e a sequência de shutdown — pontos que a spec 005 declara como congelados.

## Cenários de Usuário e Testes *(obrigatório)*

### US1 — Instrumentar os casos de uso com spans (Prioridade: P1)

Como operador, quero um span por caso de uso, para correlacionar uma requisição HTTP com as regras executadas e erros de domínio.

**Por que esta prioridade**: os spans são a única visibilidade inside do core; a spec 005 os mantém no caso de uso nesta fase.

**Teste independente**: inspeção dos quatro métodos de `internal/domain/accounts/service.go` e comparação com os testes de `service_test.go`, que executam os mesmos caminhos com mocks.

**Cenários de aceitação**:

1. **Dado** `CreateNewAccount`, **Quando** chamado, **Então** cria o span `Service::CreateNewAccount` com atributo `account_id` e o encerra com `defer span.End()`.
2. **Dados** `UpdateAccountLimits` e `UpdateAccountStatus`, **Quando** chamados, **Então** criam respectivamente `Service::UpdateAccountLimits` e `Service::UpdateAccountStatus`, com `account_id`.
3. **Dado** `ProcessEntry`, **Quando** chamado, **Então** cria `Service::ProcessEntry` com `account_id` do `Entry`.
4. **Dada** qualquer falha de repositório, regra ou atualização, **Quando** o erro ocorre, **Então** `span.SetError(err)` é chamado antes do retorno.
5. **Dada** uma falha em `ProcessEntry`, **Quando** o erro ocorre, **Então** o span recebe `AddAttributes({"entry": e})` além de `SetError`.
6. **Dado** o contexto recebido, **Quando** o span é criado, **Então** o contexto retornado por `tracer.NewSpanFromContext` é usado nas chamadas seguintes a `Repository`/`Topic`.

---

### US2 — Registrar logs estruturados das requisições e falhas (Prioridade: P1)

Como operador, quero logs de request com campos sensíveis redigidos e logs de erro nos adapters, para investigar incidentes sem expor segredos.

**Por que esta prioridade**: o middleware é a fronteira HTTP e os adapters registram detalhes que os erros de domínio não contêm.

**Teste independente**: comparação da configuração em `server.go` e das chamadas `logger.*` em `mysqldb/service.go`, `connection.go` e `message/service.go`.

**Cenários de aceitação**:

1. **Dado** `logger.InterceptorWithConfig`, **Quando** o servidor é criado, **Então** usa `MaxBodySize: 5 * 1024`, `LogRequestBody: true`, `LogResponseBody: false` e `RedactFields: ["password", "token", "credit_card"]`.
2. **Dada** uma requisição com corpo maior que 5 KiB, **Quando** o middleware processa, **Então** o corpo logado é limitado pela configuração atual.
3. **Dada** uma falha em `SaveNewAccount` ou `UpdateExistingAccount`, **Quando** `Prepare` falha, **Então** `logger.Error` registra `account`, `error` e `sql`.
4. **Dada** uma falha em `RetrieveAccountByID`, **Quando** o SQL falha, **Então** `logger.Error` registra `account` (id), `error` e `sql`.
5. **Dada** uma falha de conexão, **Quando** `Connect` falha no `sql.Open`, **Então** `logger.Fatal` registra `error` e `connection_string`.
6. **Dada** uma falha ao criar o publicador/fechar o tópico, **Quando** ocorre, **Então** o adapter registra log via `core-sdk` (`logger.Info` em `Close`).
7. **Dada** uma falha em `CreateNewAccount`, **Quando** `SaveNewAccount` falha, **Então** o `Service` registra `logger.Error` com `account` e `error`.

---

### US3 — Iniciar e encerrar o OpenTelemetry no ciclo de vida (Prioridade: P1)

Como operador, quero que o coletor receba traces do processo e que os exporters sejam fechados no shutdown, para não perder spans nem vazar conexões.

**Por que esta prioridade**: `otel.Start`/`otel.Shutdown` são chamados explicitamente em `main`; alters raising mudam a telemetria.

**Teste independente**: leitura de `cmd/main.go` e do deployment k8s, conferindo ordem de chamadas e variáveis `OTEL_*`.

**Cenários de aceitação**:

1. **Dado** `cmd/main.go`, **Quando** `main()` executa, **Então** chama `otel.Start(context.Background())`.
2. **Dado** o sinal de término, **Quando** o shutdown termina o HTTP, **Então** chama `otel.Shutdown(ctx)` com o mesmo contexto de timeout.
3. **Dada** uma falha em `otel.Shutdown`, **Quando** ocorre, **Então** é impresso `error on opentelemetry shutdown: <erro>` e o processo continua até a mensagem final.
4. **Dado** o deployment k8s, **Quando** o pod sobe, **Então** recebe `OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector-service:4317` e `OTEL_SERVICE_NAME=balances-api`.
5. **Dado** `go.mod`, **Quando** inspecionado, **Então** declara `go.opentelemetry.io/otel` e `core-sdk` como dependências diretas.

---

### US4 — Encerrar o servidor HTTP de forma graciosa (Prioridade: P1)

Como orquestrador Kubernetes, quero que o processo trate `SIGINT`/`SIGTERM` e encerre as requisições em andamento, para evitar corte abrupto.

**Por que esta prioridade**: o deployment usa `strategy: Recreate`; o timeout e a ordem de shutdown são parte do comportamento observável do pod.

**Teste independente**: leitura do trecho de shutdown em `cmd/main.go` e dos prints emitidos em cada etapa.

**Cenários de aceitação**:

1. **Dado** `signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)`, **Quando** um desses sinais chega, **Então** o canal `quit` é liberado e `main` prossegue.
2. **Dado** o início do shutdown, **Quando** a mensagem é emitida, **Então** é impresso `starting graceful shutdown...`.
3. **Dado** `context.WithTimeout(context.Background(), 25*time.Second)`, **Quando** o shutdown começa, **Então** HTTP e otel usam o mesmo `ctx`, com `defer cancel()`.
4. **Dado** `s.Shutdown(ctx)`, **Quando** retorna sucesso, **Então** é impresso `HTTP server finished with success`; em erro, `error on HTTP server shutdown: <erro>`.
5. **Dado** o fim do fluxo, **Quando** todas as etapas terminam, **Então** é impresso `graceful shutdown done`.
6. **Dado** `Server.Shutdown(ctx)`, **Quando** chamado, **Então** delega para `echo.Echo.Shutdown(ctx)`.

---

### US5 — Manter o health check como liveness do processo (Prioridade: P2)

Como Kubernetes, quero consultar uma rota leve para saber se o processo está no ar, sem depender de MySQL ou SNS.

**Por que esta prioridade**: é o único mecanismo de health existente; a spec 005 propõe um `HealthChecker` injetado para `/health/ready`.

**Teste independente**: `GET /` — contrato completo especificado na US6 da spec 002.

**Cenários de aceitação**:

1. **Dado** o servidor iniciado, **Quando** `GET /` é chamada, **Então** responde `200` com o corpo definido na spec 002.
2. **Dada** a rota de health, **Quando** avaliada nesta spec, **Então** não consulta `HealthChecker`, MySQL, SNS ou SQS.
3. **Dado** um processo em shutdown, **Quando** o HTTP server é desligado, **Então** a rota deixa de responder junto com as demais.

---

### Casos Limite

- `otel.Start` é chamado **depois** de iniciar a goroutine do servidor; o comportamento atual é preservado.
- `config.New(config.WithAppPort(5002), config.WithMysqlDBName("balances"))` é chamado dentro da goroutine do servidor — desvio registrado; a spec 005 move essa chamada para a raiz.
- Mensagens de shutdown usam `fmt.Println`/`fmt.Printf`, enquanto o restante da aplicação usa `logger`; mistura atual congelada.
- Não há `/health/ready`, nem verificação de MySQL/SNS no health check.
- Não há correlation ID ou trace ID injetado em cabeçalhos de resposta; a correlação depende do middleware/coletor.
- `Service` cria spans mesmo em testes unitários; isso é tolerado pelo `core-sdk` e faz parte do baseline.
- `logger.Fatal` em `Connect()` e `logger.Fatal` em `main` seguidos de `os.Exit(1)` atualmente coexistem; nenhum dos dois é alterado aqui.
- A ordem de shutdown é HTTP → otel; não há fechamento explícito de conexão MySQL ou publicador SNS no `main`.
- Span names usam `::` (`Service::ProcessEntry`); são tratados como contrato de telemetria.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: Cada método público de `Service` DEVE abrir e fechar um span com o nome `Service::<Nome>` e atributo `account_id`.
- **FR-002**: Falhas nos casos de uso DEVEM marcar o span com `SetError`.
- **FR-003**: Falhas em `ProcessEntry` DEVEM adicionar o atributo `entry` ao span.
- **FR-004**: O middleware HTTP DEVE manter os valores atuais de limite de corpo, captura de request e redação de campos sensíveis.
- **FR-005**: Logs de adapters DEVEM permanecer estruturados, com os campos hoje registrados.
- **FR-006**: `main()` DEVE chamar `otel.Start` e `otel.Shutdown` na ordem atual.
- **FR-007**: O deployment k8s DEVE continuar fornecendo `OTEL_EXPORTER_OTLP_ENDPOINT` e `OTEL_SERVICE_NAME`.
- **FR-008**: `main()` DEVE tratar `SIGINT` e `SIGTERM` e usar timeout de 25 segundos para o shutdown.
- **FR-009**: A ordem HTTP → otel e as mensagens impressas no shutdown NÃO PODEM mudar.
- **FR-010**: `Server.Shutdown(ctx)` DEVE delegar ao `Shutdown` do Echo.
- **FR-011**: A rota `GET /` DEVE permanecer como liveness leve, conforme spec 002.
- **FR-012**: Nenhum decorator de tracing/logging DEVE ser introduzido nesta spec; spans permanecem no core.
- **FR-013**: A spec 005 PODE mover arquivos e injetar dependências, mas NÃO DEVE alterar nomes de span, atributos, configuração de log nem sequência de shutdown.
- **FR-014**: Ausência de `/health/ready`, de correlation ID e de fechamento explícito de adapters no `main` é o estado atual documentado.

### Entidades Principais

- **Span `Service::*`**: unidade de telemetria dos casos de uso.
- **logger.InterceptorConfig**: configuração do middleware HTTP.
- **otel.Start/Shutdown**: ciclo de vida do coletor OpenTelemetry.
- **quit channel / signal.Notify**: tratamento de sinais do sistema operacional.
- **Server.Shutdown**: desligamento gracioso do Echo.
- **HealthCheck**: liveness HTTP definida na spec 002.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: Os quatro nomes de span e o atributo `account_id` permanecem idênticos aos de `service.go`.
- **SC-002**: `SetError` aparece em todos os caminhos de erro dos quatro casos de uso.
- **SC-003**: A configuração do interceptador de log permanece com os quatro valores atuais.
- **SC-004**: O deploy k8s continua exportando para `otel-collector-service:4317` com `OTEL_SERVICE_NAME=balances-api`.
- **SC-005**: `SIGTERM` produz as três mensagens de shutdown na ordem atual, com timeout de 25s.
- **SC-006**: `GET /` continua respondendo `200` com o corpo da spec 002 e sem consultar dependências.
- **SC-007**: A spec 005 pode mover `Service` e handlers sem alterar nenhum item medido acima.

## Evidência de Implementação (engenharia reversa)

| Comportamento | Arquivo(s) | Histórico |
|---|---|---|
| Spans nos casos de uso | `internal/domain/accounts/service.go` | `bb5f9db`, `ecbf7c2`, `27ea5b2`, `16ba535` |
| Middleware e health HTTP | `internal/infra/rest/server/server.go` | `a63fc2e`, `47fa6fe`, `dd57b8c` |
| Logs em adapters | `internal/infra/database/mysqldb/*.go`, `internal/infra/message/service.go` | `ee8b282`, `ecbf7c2` |
| Início/fim do processo | `cmd/main.go` | `bb5f9db`, `eab8468`, `ecbf7c2` |
| Coletor no k8s | `scripts/k8s/app-deployment.yaml` | `2298435`, `ecbf7c2` |
| Dependências de telemetria | `go.mod` | `8b5a8cf`, `ecbf7c2` |

## Suposições

- Nomes de span e mensagens de log são considerados contrato de telemetria, ainda que não sejam contrato HTTP.
- O `core-sdk` absorve a configuração de exportação via variáveis de ambiente; este repo não configura exporters manualmente.
- Os testes unitários atuais não validam spans; a verificação desta spec é por inspeção e por comparação com a spec 005.
- A spec 004 não altera `cmd/main.go`; a composição da raiz é tema da spec 005.
