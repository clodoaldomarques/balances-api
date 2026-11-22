# Tarefas: `balances-api` — Observabilidade, Health e Shutdown

**Repo**: `github.com/clodoaldomarques/balances-api` | **Ordem**: 4 de 5 (sequência de specs neste repo; a 005 é canônica da família)

**Branch**: `004-observabilidade-e-shutdown`

**Entrada**: [spec.md](spec.md) e [plan.md](plan.md) deste diretório.

**Nota**: especificação **retroativa**. `[x]` = já implementado e verificável em `16ba535`; `[ ]` = lacuna identificada. Marcos `T001`-`T014` e evidências `BA-001`-`BA-007`.

**Formato**: `[ID] [P] [US] Descrição com caminho exato do arquivo`.

## Fase 1: Spans nos casos de uso

**Objetivo**: rastrear cada caso de uso com um span nomeado e erro marcado.

**Teste independente**: leitura dos quatro métodos de `internal/domain/accounts/service.go`.

- [x] T001 [US1] Criar `Service::CreateNewAccount` com `account_id` e `defer span.End()`
- [x] T002 [US1] Criar `Service::UpdateAccountLimits` e `Service::UpdateAccountStatus` com `account_id`
- [x] T003 [US1] Criar `Service::ProcessEntry` com `account_id` do `Entry`
- [x] T004 [US1] Chamar `SetError` em todos os caminhos de falha; adicionar `AddAttributes({"entry": e})` em `ProcessEntry`
- [x] BA-001 `tracer.NewSpanFromContext` e `span.SetError` em `service.go` (`bb5f9db`, `ecbf7c2`)

**Checkpoint**: quatro spans nomeados, todos com `account_id` e `SetError` nos erros.

---

## Fase 2: Logs estruturados e middleware HTTP

**Objetivo**: capturar request body com limite, redigir segredos e registrar falhas nos adapters.

- [x] T005 [US2] Instalar `logger.InterceptorWithConfig` em `server.go` com os valores atuais
- [x] T006 [US2] Registrar erros de SQL/conexão em `mysqldb` e log de `Close` em `message`
- [x] T007 [US2] Registrar `logger.Error` em `CreateNewAccount` quando `SaveNewAccount` falha
- [x] BA-002 `MaxBodySize: 5 * 1024`, `LogRequestBody: true`, `LogResponseBody: false`, `RedactFields: ["password", "token", "credit_card"]`
- [x] BA-003 `logger.Error`/`logger.Fatal` em `mysqldb/{connection,service}.go` e `logger.Info` em `message/service.go`

**Checkpoint**: configuração do interceptador idêntica à da spec 004; campos sensíveis redigidos.

---

## Fase 3: Ciclo de vida do OpenTelemetry

**Objetivo**: iniciar e encerrar o coletor no `main` e exportar para o coletor do k8s.

- [x] T008 [US3] Chamar `otel.Start(context.Background())` em `cmd/main.go`
- [x] T009 [US3] Chamar `otel.Shutdown(ctx)` após o shutdown HTTP
- [x] T010 [US3] Definir `OTEL_EXPORTER_OTLP_ENDPOINT` e `OTEL_SERVICE_NAME` no deployment k8s
- [x] BA-004 `otel.Start`/`otel.Shutdown` em `main` (`ecbf7c2`)
- [x] BA-005 env vars OTEL em `scripts/k8s/app-deployment.yaml`

**Checkpoint**: coletor configurado e desligado ao final do processo.

---

## Fase 4: Shutdown gracioso

**Objetivo**: tratar sinais do SO e encerrar HTTP e otel com timeout controlado.

- [x] T011 [US4] Registrar `signal.Notify(quit, SIGINT, SIGTERM)` e aguardar `<-quit`
- [x] T012 [US4] Criar `context.WithTimeout(..., 25*time.Second)` com `defer cancel()`
- [x] T013 [US4] Chamar `s.Shutdown(ctx)` → `echo.Echo.Shutdown` e imprimir mensagens atuais
- [x] T014 [US5] Manter `GET /` como liveness leve, conforme spec 002
- [x] BA-006 `cmd/main.go`: `starting graceful shutdown...`, `HTTP server finished with success`, `error on HTTP server shutdown: ...`, `graceful shutdown done`
- [x] BA-007 `Server.Shutdown(ctx)` em `internal/infra/rest/server/server.go`

**Checkpoint**: `SIGTERM` encerra o processo na ordem HTTP → otel em até 25 s.

---

## Dependências e Ordem de Execução

- US1 depende dos casos de uso da spec 001; US2 e US5 dependem do `Server` da spec 002.
- US3 e US4 usam os adapters da spec 003 apenas como origem dos logs — não os modificam.
- Lacunas conhecidas (`/health/ready`, correlation ID, `config.New` na raiz, decorators) pertencem à spec 005 ou a spec futura.
