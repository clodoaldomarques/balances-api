# Especificação da Funcionalidade: Adoção da Arquitetura Hexagonal — `balances-api`

**Branch da funcionalidade**: `005-hexagonal-adapters`

**Criado**: 2026-11-22

**Repo**: `github.com/clodoaldomarques/balances-api`

**Status**: Proposta — especificação **prescritiva** (estado desejado), ainda não implementada

**Entrada**: engenharia reversa deste repo (`internal/domain/accounts`, `internal/infra/**`, `cmd/main.go`) e convenção da família ledger/balances registrada neste diretório.

> **Escopo**: esta é a spec 005 aplicada a `balances-api`. Os IDs `US1`-`US5`, `FR-001`..`FR-014` e `SC-001`..`SC-007` são idênticos nos repos da família para rastreabilidade; o conteúdo abaixo descreve **somente este repo**. Nenhuma mudança de comportamento externo está autorizada.

## Contexto e Motivação

`balances-api` concentra entidades, regras, caso de uso, portas e mocks em um único pacote `internal/domain/accounts`: `service.go` (o caso de uso) convive com `account.go`, `balances.go`, `entry.go`, etc., e `interfaces.go` expõe `Repository`, `Topic` e `Event` com `go:generate`. Os handlers de `internal/infra/rest/accounts` constroem MySQL, SNS e `Service` a cada requisição, e `cmd/main.go` hoje apenas inicializa o servidor e chama `config.New()` dentro de uma goroutine.

Isso dificulta trocar MySQL ou SNS sem tocar nas regras, testar handlers sem banco e manter a mesma convenção dos demais repos. A migração é estrutural: separar `domain` de `application` e `ports`, mover handlers/adapters para `internal/adapters`, e fazer de `cmd/main.go` a raiz de composição. Regras de negócio, contrato HTTP, contrato SNS e esquema MySQL permanecem intactos.

## Cenários de Usuário e Testes *(obrigatório)*

### US1 — Trocar adaptador de saída sem alterar casos de uso (Prioridade: P1)

Como mantenedor de `balances-api`, quero substituir MySQL, SNS ou o publisher de eventos por outro adaptador ou fake sem modificar `service.go`.

**Por que esta prioridade**: é a garantia central da arquitetura hexagonal e permite testar regras de saldo/limites sem banco.

**Teste independente**: executar a mesma suíte de `service_test.go` com um fake in-memory de `AccountRepository`/`AccountEventPublisher` e com os adapters reais; ambos devem satisfazer os mesmos contratos.

**Cenários de aceitação**:

1. **Dado** `internal/core/accounts/**`, **Quando** inspecionar seus imports, **Então** não há import de `internal/adapters`, driver MySQL, AWS SDK, Echo nem `config`.
2. **Dado** um fake de `AccountRepository`, **Quando** injetado no `Service`, **Então** abertura, atualização, consulta e `SaveEntryAndUpdateAccount` se comportam como hoje.
3. **Dado** o adapter MySQL real, **Quando** compilado, **Então** satisfaz integralmente `AccountRepository`.
4. **Dado** um método novo na porta, **Quando** algum adapter não o implementar, **Então** o compilador falha — sem fallback dinâmico.

---

### US2 — Adicionar adaptador de entrada sem duplicar regra (Prioridade: P1)

Como mantenedor, quero que os handlers HTTP e um futuro consumer chamem o mesmo caso de uso pela porta `AccountsUseCase`.

**Por que esta prioridade**: hoje os handlers constroem `Service` diretamente; um segundo adaptador precisaria conhecer a implementação concreta e poderia replicar validação de saldo/limites.

**Teste independente**: um adaptador de entrada alternativo invoca `AccountsUseCase` e verifica o mesmo resultado do handler HTTP, com o mesmo `Service`.

**Cenários de aceitação**:

1. **Dado** o handler HTTP e um segundo adaptador, **Quando** ambos chamam o caso de uso, **Então** utilizam a mesma implementação e as mesmas regras.
2. **Dado** um DTO de qualquer adaptador de entrada, **Quando** convertido para entidade, **Então** a validação permanece em `domain`.
3. **Dado** um teste de handler, **Quando** ele injeta um fake de `AccountsUseCase`, **Então** não são necessários MySQL, SNS ou LocalStack.

---

### US3 — Compor dependências uma única vez na raiz (Prioridade: P1)

Como operador/mantenedor, quero que `cmd/main.go` construa adapters e casos de uso uma única vez e os injete nos handlers.

**Por que esta prioridade**: cada handler de `accounts` hoje cria MySQL + SNS + `Service` por request; isso espalha composição e impede o controle do ciclo de vida (inclusive o `config.New(config.WithAppPort(5002), ...)` chamado dentro da goroutine).

**Teste independente**: subir o servidor com adapters fake e verificar uma instância de cada porta por processo e fechamento explícito no graceful shutdown.

**Cenários de aceitação**:

1. **Dado** `cmd/main.go`, **Quando** a aplicação inicia, **Então** cria config uma vez, conexão MySQL, publicador SNS, `Service`, handlers e servidor, injetando as dependências.
2. **Dado** qualquer handler, **Quando** uma requisição chega, **Então** ele não chama construtores de adapter nem `NewService`.
3. **Dado** `/health/ready` (ou rota de health equivalente), **Quando** chamado, **Então** consulta `HealthChecker` injetado.
4. **Dado** o sinal de shutdown, **Quando** o graceful shutdown termina, **Então** adapters com `Close()` são fechados pela raiz, preservando o fluxo atual.

---

### US4 — Preservar contrato externo e levantar guarda arquitetural (Prioridade: P2)

Como revisor, quero garantir que a migração não altere API, eventos ou modelo de dados, e que a regra de dependência continue válida.

**Por que esta prioridade**: a mudança é estrutural; qualquer diferença observável seria um incidente disfarçado de refatoração.

**Teste independente**: `go test ./...`, `go build ./...`, `go vet ./...`, `make arch-check` e comparação das respostas HTTP atuais de `accounts` e health.

**Cenários de aceitação**:

1. **Dadas** as rotas atuais, **Quando** a migração termina, **Então** paths, status codes e corpo das respostas permanecem idênticos.
2. **Dado** o contrato SNS, **Quando** comparado ao estado atual, **Então** tópico, payload de `Event` e mapeamento `EventType()`/`EventData()` ficam congelados.
3. **Dado** um import de `core` para `adapters`, **Quando** `make arch-check` roda, **Então** o comando falha com arquivo e linha infratores.
4. **Dado** o `git diff`, **Quando** revisado, **Então** as entidades movidas para `internal/core/accounts/domain` não têm regra alterada.

---

### US5 — Remover dependências órfãs e mocks obsoletos (Prioridade: P2)

Como mantenedor, quero que nenhuma porta sem uso ou mock desatualizado sobreviva à separação de pacotes.

**Por que esta prioridade**: este repo não tem `gringotts`/`ProgramAPI`/`ollama`, mas `mock.go` nasce de `internal/domain/accounts/interfaces.go` e precisa ser regenerado no novo pacote de portas.

**Teste independente**: `go mod tidy` e busca por `gringotts`, `Griphook`, `ProgramAPI`, `ollama` e por mocks fora de `ports/` não retornam ocorrências.

**Cenários de aceitação**:

1. **Dado** `internal/domain/accounts/mock.go`, **Quando** a migração termina, **Então** ele existe apenas como `internal/core/accounts/ports/mock.go`, gerado a partir das portas finais.
2. **Dado** `interfaces.go`, **Quando** movido para `ports/outbound.go`, **Então** a diretiva `go:generate` e o pacote do mock são ajustados.
3. **Dado** `go.mod`, **Quando** `go mod tidy` roda, **Então** dependências não usadas desaparecem.
4. **Dada** alguma funcionalidade HTTP, **Quando** comparada ao estado anterior, **Então** não houve alteração.

---

### Casos Limite

- Conexões MySQL e publicadores passam a ser compartilhados pela vida do processo; confirmar thread safety antes da Fase 3.
- `config.New()` continua `sync.Once`, mas as opções (`WithAppPort(5002)`, `WithMysqlDBName("balances")`) devem ser aplicadas uma vez na raiz, não dentro da goroutine do servidor.
- `tracer`/`logger` permanecem dentro do core nesta spec; decorators são fase opcional futura.
- `Event` implementa `EventType()`/`EventData()` e é contrato de domínio: permanece no core, não vira DTO de adaptador.
- Bugs já documentados em specs da família não são corrigidos aqui.
- Este repo não possui collection Postman versionada; a verificação de contrato usa as rotas atuais e os testes existentes. As specs descritivas `001`-`004` foram reconstruídas por engenharia reversa em 2026-11-22 e estão neste mesmo diretório.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: O core (`internal/core/accounts/**`) NÃO DEVE importar `internal/adapters`, driver MySQL, AWS SDK, Echo nem `config`.
- **FR-002**: As portas DEVEM ser declaradas em `internal/core/accounts/ports` e implementadas pelos adapters; a dependência aponta sempre do adapter para o core.
- **FR-003**: O caso de uso DEVE ser exposto por `AccountsUseCase` no core, consumido pelos handlers HTTP e por futuros adaptadores de entrada.
- **FR-004**: A raiz de composição (`cmd/main.go`) DEVE instanciar conexão, publicador, `Service`, handlers e servidor uma única vez, injetando as dependências.
- **FR-005**: Os handlers NÃO DEVEM construir adapters nem casos de uso; recebem `AccountsUseCase` por construtor.
- **FR-006**: Health checks DEVEM ser expostos como porta `HealthChecker` injetada e consumida pelo adaptador HTTP.
- **FR-007**: O mapeamento de erros de domínio para status HTTP (`400`/`404`) DEVE ficar em `adapters/inbound/http/errors.go`.
- **FR-008**: As portas DEVEM ter nomes que expressem intenção: `AccountRepository`, `AccountEventPublisher`, `AccountsUseCase`, `HealthChecker` — em vez de `Repository`/`Topic`.
- **FR-009**: Este repo não contém `ProgramAPI`/`gringotts`/`ollama`; `mock.go` DEVE ser regenerado no pacote de portas e nenhuma porta órfã DEVE permanecer.
- **FR-010**: O comportamento observável NÃO PODE mudar: rotas, status, JSON, validações, esquema MySQL, tópico SNS e contrato de evento.
- **FR-011**: DEVE existir `make arch-check` neste repo, executado antes do merge.
- **FR-012**: O `docs/specs/README.md`, `platform.md` e a documentação de arquitetura afetada DEVEM ser atualizados na mesma mudança de código.
- **FR-013**: A limpeza de mocks/portas NÃO DEVE alterar funcionalidade observável.
- **FR-014**: Este repo DEVE manter um único `spec.md`, `plan.md` e `tasks.md` coerentes com o layout alvo, mesmo que a implementação ocorra em PRs separados.

### Entidades Principais

- **Porta de entrada (driving port)**: `AccountsUseCase` no core, com os métodos públicos atuais de `Service`.
- **Porta de saída (driven port)**: `AccountRepository`, `AccountEventPublisher`, `HealthChecker` no core; `Event` permanece como contrato de domínio.
- **Adapter de entrada**: handlers HTTP de `accounts` + `server` + mapeador de erros.
- **Adapter de saída**: `adapters/outbound/mysql` e `adapters/outbound/sns`.
- **Composition root**: `cmd/main.go`; único lugar que conhece adapters concretos e configuração.
- **Guarda de arquitetura**: `scripts/check-architecture.sh`, exposta por `make arch-check`.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: `go test ./...`, `go build ./...` e `go vet ./...` passam neste repo antes e depois da migração.
- **SC-002**: Zero imports de `internal/adapters` a partir de `internal/core` neste repo.
- **SC-003**: Zero chamadas a construtores de adapter fora de `cmd/**` e dos próprios adapters.
- **SC-004**: Diff de `internal/core/accounts/domain` restrito a movimento de pacote/imports — nenhuma regra alterada, especialmente a separação entre `Service` e entidades.
- **SC-005**: Nenhuma diferença nas respostas HTTP de `accounts` nem no contrato SNS.
- **SC-006**: `go mod tidy` sem dependências órfãs e sem ocorrências de `gringotts`/`ProgramAPI`/`Griphook`/`ollama` neste repo.
- **SC-007**: `make arch-check` falha quando um import proibido é introduzido artificialmente e passa no estado final.

## Suposições

- Conexões MySQL e clientes AWS do `core-sdk` são seguros para reutilização concorrente; confirmar na Fase 3.
- O contrato externo atual é o correto, inclusive desvios já registrados; correções viram specs separadas.
- Este repo é um módulo Go independente (`go.mod` próprio).
- Toda a documentação necessária para este repo está neste diretório; não há dependência de links externos.
- Não há CI; a guarda arquitetural é executada por comando local e pelo revisor.
