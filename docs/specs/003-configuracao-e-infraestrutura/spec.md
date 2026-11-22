# Especificação da Funcionalidade: Configuração, Persistência MySQL e Mensageria — `balances-api`

**Branch da funcionalidade**: `003-configuracao-e-infraestrutura`

**Criado**: 2026-11-22

**Repo**: `github.com/clodoaldomarques/balances-api`

**Status**: Implementada (derivada do código em `16ba535`) — especificação **retroativa e descritiva**, reconstruída por engenharia reversa de `main`

**Entrada**: `config/config.go`, `internal/infra/database/mysqldb/**`, `internal/infra/message/service.go`, `scripts/**`, `.env` (local, não versionado) e histórico Git (`866dd08`, `8398f3e`, `7245533`, `4c2e592`, `8b5a8cf`, `d8b5ad3`, `eab8468`, `8fb2ccc`, `ab42c68`).

> **Escopo**: documenta como configuração, MySQL, SNS/SQS e infraestrutura local estão implementados hoje. Fazem parte desta spec o esquema SQL e os contratos de persistência/publicação congelados pela spec 005. Nenhuma mudança de comportamento é autorizada.

## Contexto e Motivação

`config/config.go` centraliza variáveis de ambiente em um singleton `sync.Once` e expõe options (`WithAppPort`, `WithMysqlDBName`, etc.). `internal/infra/database/mysqldb` implementa o `Repository` da spec 001 com `database/sql`, quatro SQL statements e uma transação para gravar entrada + atualizar conta. `internal/infra/message` implementa `Topic` usando o publisher SNS do `core-sdk`, com envelope `sns.Event` (uuid, tipo, dados, data). `scripts/terraform` provê tópico SNS, fila SQS e a assinatura; `scripts/k8s` cria o schema, injeta secrets e expõe a porta; `scripts/docker` e `Makefile` fecham o ciclo local.

Esta spec dá rastreabilidade a esse conjunto — inclusive às divergências atuais de porta e às limitações conhecidas da transação — para que a refatoração da spec 005 não altere contrato de dados nem de mensageria.

## Cenários de Usuário e Testes *(obrigatório)*

### US1 — Carregar configuração uma única vez a partir do ambiente (Prioridade: P1)

Como operador, quero configurar o serviço por variáveis de ambiente, para executá-lo em Minikube/LocalStack sem recompilar.

**Por que esta prioridade**: todos os adapters dependem de `config.New()`; mudar defaults ou chaves quebra o deploy.

**Teste independente**: executar `config.New()` com e sem `APP_PORT`/`MYSQL_*` definidos e conferir os valores efetivos e o DSN montado.

**Cenários de aceitação**:

1. **Dado** `config.New()` sem opções e sem env, **Quando** chamado, **Então** usa os defaults atuais: `AppPort=5000`, `MYSQL_HOST=192.168.49.2`, `MYSQL_PORT=30001`, `MYSQL_DATABASE=balances`, demais strings vazias.
2. **Dado** `sync.Once`, **Quando** `New()` é chamado várias vezes, **Então** os valores padrão são lidos do ambiente apenas na primeira chamada.
3. **Dadas** options como `WithAppPort(5002)` e `WithMysqlDBName("balances")`, **Quando** passadas, **Então** são aplicadas à instância retornada.
4. **Dado** `GetMySQLConnectionString()`, **Quando** chamado, **Então** devolve `<user>:<pass>@tcp(<host>:<port>)/<db>?charset=utf8&parseTime=True&loc=Local`.
5. **Dados** os methods `Region()`, `Address()`, `AccessKeyID()`, `SecretAccessKey()` e `TopicARN()`, **Quando** chamados, **Então** devolvem os campos AWS correspondentes.

---

### US2 — Persistir contas e entradas no MySQL (Prioridade: P1)

Como caso de uso, quero gravar contas e lançamentos com o esquema atual, para manter o estado financeiro consistente.

**Por que esta prioridade**: os SQL statements e a transação são o contrato de dados congelado pela spec 005.

**Teste independente**: inspeção dos statements em `service.go` e do schema aplicado pelo initContainer `init-db` via ConfigMap `balances-schema`.

**Cenários de aceitação**:

1. **Dado** `mysqldb.Connect()`, **Quando** chamado, **Então** abre `sql.Open("mysql", DSN)` e executa `Ping`; em falha de ping, fecha a conexão e devolve o erro.
2. **Dado** `SaveNewAccount`, **Quando** executado, **Então** usa `INSERT_ACCOUNT` com os oito campos `account_id`, `org_id`, `limits`, `balances`, `created_at`, `updated_at`, `status`, `version`.
3. **Dado** `UpdateExistingAccount`, **Quando** executado, **Então** usa `UPDATE_ACCOUNT` com `WHERE account_id = ? AND org_id = ?`.
4. **Dado** `RetrieveAccountByID`, **Quando** executado, **Então** usa `SELECT_ACCOUNT` com a mesma chave composta e converte a linha com `toEntity()`.
5. **Dado** `SaveEntryAndUpdateAccount`, **Quando** executado, **Então** inicia uma transação, insere em `entries` (`INSERT_ENTRIES`) e atualiza `accounts` (`UPDATE_ACCOUNT`) antes de `Commit`.
6. **Dado** o schema, **Quando** o initContainer roda, **Então** cria `accounts`, `entries` e `daily_balances` com as colunas atuais do ConfigMap.
7. **Dados** `limits`, `balances` e `impacts`, **Quando** gravados, **Então** são serializados como JSON via `commons.DecimalMap`/`encoding/json`.

---

### US3 — Publicar eventos de domínio no SNS (Prioridade: P1)

Como worker, quero consumir eventos publicados no tópico `balances-sns-topic`, para manter o lado de leitura atualizado.

**Por que esta prioridade**: o tópico, o envelope e o payload são contrato externo; a spec 005 os congela explicitamente.

**Teste independente**: inspeção de `message/service.go`, `config.TopicARN()`, `.env` e `scripts/terraform/sns.tf`.

**Cenários de aceitação**:

1. **Dado** `message.NewTopic(ctx)`, **Quando** construído, **Então** cria o publisher com `sns.NewPublisher(ctx, config.New())` do `core-sdk`.
2. **Dado** `Topic.Emit(ctx, evt)`, **Quando** chamado, **Então** monta `sns.Event` com `EventID = uuid.New()`, `EventType = evt.EventType()`, `EventData = evt.EventData()` e `EventDate = time.Now()`.
3. **Dado** o retorno de `Emit`, **Quando** o publisher falha, **Então** o erro é devolvido ao chamador (que hoje o descarta no `Service` — ver spec 001).
4. **Dado** `Topic.Close(ctx)`, **Quando** chamado, **Então** apenas registra log `ending topic connection`; não há recurso adicional a fechar.
5. **Dado** o tópico configurado, **Quando** o deploy roda, **Então** é usado o ARN `arn:aws:sns:us-east-1:000000000000:balances-sns-topic` (valor de `.env` e do deployment k8s).

---

### US4 — Provisionar mensageria e ambiente local (Prioridade: P1)

Como desenvolvedor, quero subir LocalStack, MySQL e o tópico/fila com comandos reproduzíveis, para validar o fluxo assíncrono localmente.

**Por que esta prioridade**: o repo não tem CI; o ambiente local é a única forma de verificar a integração ponta a ponta.

**Teste independente**: `make terraform` (após LocalStack disponível em `192.168.67.2:30002`) e `kubectl apply -f scripts/k8s/`.

**Cenários de aceitação**:

1. **Dado** `scripts/terraform/sns.tf`, **Quando** aplicado, **Então** cria `aws_sns_topic.balances-sns-topic` e sua política com `account-id = "000000000000"`.
2. **Dado** `scripts/terraform/sqs.tf`, **Quando** aplicado, **Então** cria `balances-sqs-queue` e uma assinatura SNS→SQS entre tópico e fila.
3. **Dado** `scripts/terraform/provider.tf`, **Quando** aplicado, **Então** usa provider AWS `~> 5.0` apontando para o LocalStack (`192.168.49.2:30002`) com credenciais de teste.
4. **Dado** `scripts/k8s/app-deployment.yaml`, **Quando** aplicado, **Então** o initContainer espera o MySQL, executa o schema e injeta credenciais via secrets `mysql-credentials` e `aws-credentials`.
5. **Dado** `scripts/k8s/app-service.yaml`, **Quando** aplicado, **Então** expõe `LoadBalancer` na porta 5002, `targetPort` 5002 e `nodePort` 31002.
6. **Dado** o `Makefile`, **Quando** `make apply`/`make destroy`/`make test` são executados, **Então** executam terraform + kubectl, remoção dos recursos ou `go test ./... -coverprofile cover.out`.

---

### US5 — Iolar adapters de infraestrutura do domínio (Prioridade: P2)

Como mantenedor, quero que MySQL e SNS implementem interfaces declaradas no domínio, para que a regra financeira não conheça detalhes de driver ou SDK.

**Por que esta prioridade**: é o contrato que a spec 005 transforma em portas explícitas.

**Teste independente**: compilação de `internal/domain/accounts/service_test.go` com `NewMockRepository`/`NewMockTopic`, sem MySQL nem AWS.

**Cenários de aceitação**:

1. **Dado** `Repository` (`SaveNewAccount`, `UpdateExistingAccount`, `RetrieveAccountByID`, `SaveEntryAndUpdateAccount`), **Quando** `mysqldb.Repository` é compilado, **Então** satisfaz a interface.
2. **Dado** `Topic` (`Emit`), **Quando** `message.Topic` é compilado, **Então** satisfaz a interface.
3. **Dado** `Service`, **Quando** testado, **Então** recebe mocks gerados de `interfaces.go` — nenhum banco ou AWS é necessário.
4. **Dado** `internal/domain/accounts`, **Quando** inspecionado, **Então** não importa `mysqldb` nem `message`; a dependência aponta do adapter para o domínio.

---

### Casos Limite

- `config.New()` lê o ambiente uma vez (`sync.Once`), mas **aplica options em toda chamada** — comportamento atual registrado; a spec 005 exige aplicação única na raiz.
- `BalancesSQSQueue` existe em `Config` e é lido de `BALANCES_SQS_QUEUE`, mas não é usado por este repo; a fila é provisionada para o worker.
- `Connect()` chama `logger.Fatal` e, em seguida, `return nil, err`; o retorno é considerado código morto na prática.
- `NewRepository(ctx)` devolve `&Repository{ctx: ctx}` com `db == nil` quando a conexão falha; um `Close()` posterior pode falhar — congelado aqui.
- `SaveEntryAndUpdateAccount` usa `r.ctx` (capturado no construtor) em `BeginTx`, ignora o erro de `tx.Commit()` e não fecha o segundo `statement` preparado.
- `mysqlDB` não define pool (`SetMaxOpenConns`/`SetConnMaxLifetime`); conexões são abertas e fechadas por request.
- `scripts/sql/ddl_create_database.sql` usa `IF NOT EXISTS CREATE TABLE` (ordem histórica); o schema efetivamente aplicado no k8s é o do ConfigMap, com `CREATE TABLE IF NOT EXISTS`.
- `daily_balances` é criado, mas não escrito por esta API — pertence ao `balances-worker`.
- Terraform e deployment usam endpoint/credenciais fixos de LocalStack; não há segredo real versionado.
- Docker expõe 5000, enquanto a opção em `main` e o Service k8s usam 5002 — divergência atual documentada, não corrigida nesta spec.

## Requisitos *(obrigatório)*

### Requisitos Funcionais

- **FR-001**: `config.New()` DEVE ler o ambiente uma única vez via `sync.Once`, com os defaults atuais.
- **FR-002**: As options existentes DEVEM continuar disponíveis e aplicar seus valores à instância retornada.
- **FR-003**: `GetMySQLConnectionString()` DEVE manter o formato DSN atual, incluindo `charset=utf8&parseTime=True&loc=Local`.
- **FR-004**: `mysqldb.Repository` DEVE implementar integralmente `accounts.Repository`.
- **FR-005**: Os quatro statements SQL DEVEM permanecer com o texto e os parâmetros atuais.
- **FR-006**: Gravação de entrada e atualização de conta DEVEM ocorrer na mesma transação.
- **FR-007**: O esquema MySQL DEVE conter `accounts`, `entries` e `daily_balances` com as colunas atuais.
- **FR-008**: `message.Topic` DEVE implementar `accounts.Topic` usando o publisher SNS do `core-sdk`.
- **FR-009**: O envelope publicado DEVE conter `EventID`, `EventType`, `EventData` e `EventDate` conforme `message/service.go`.
- **FR-010**: O ARN do tópico DEVE ser configurável por `BALANCES_SNS_TOPIC`/`WithSnsAccountTopic` e manter o valor atual nos ambientes versionados.
- **FR-011**: Terraform DEVE provisionar tópico SNS, política, fila SQS e assinatura SNS→SQS.
- **FR-012**: O deployment k8s DEVE criar o schema no initContainer e injetar credenciais por secrets.
- **FR-013**: `Makefile`, Dockerfile e manifests DEVEM permanecer executáveis sem mudança de ports, nomes de imagem alvos ou comandos.
- **FR-014**: Os desvios listados em "Casos Limite" NÃO DEVEM ser corrigidos nesta spec; correção exige spec própria.

### Entidades Principais

- **Config / Option**: singleton de configuração e funções de ajuste.
- **mysqldb.Repository / Connection**: implementação MySQL de `accounts.Repository`.
- **Account / Entry table types**: mapeamento entre entidades e linhas SQL (`tables.go`).
- **message.Topic**: implementação SNS de `accounts.Topic`.
- **sns.Event**: envelope de mensagem do `core-sdk`.
- **Terraform SNS/SQS, ConfigMap, Deployment, Service, Dockerfile, Makefile**: infraestrutura versionada.

## Critérios de Sucesso *(obrigatório)*

### Resultados Mensuráveis

- **SC-001**: `go build ./...` e `go vet ./...` passam sem alterar `config/`, `internal/infra/database/**` ou `internal/infra/message/**`.
- **SC-002**: O DSN gerado por `GetMySQLConnectionString()` é idêntico ao usado por `Connect()`.
- **SC-003**: Os quatro statements SQL e a transação de `SaveEntryAndUpdateAccount` permanecem byte a byte equivalentes.
- **SC-004**: O envelope SNS e os três `EventType()` da spec 001 permanecem inalterados.
- **SC-005**: `terraform plan` contra LocalStack não propõe mudança no tópico, na fila ou na assinatura.
- **SC-006**: Os manifests k8s continuam aplicáveis com os mesmos secrets, portas e nome de imagem.
- **SC-007**: A spec 005 pode trocar a implementação dos adapters sem alterar SQL, tópico, payload ou configuração observável.

## Evidência de Implementação (engenharia reversa)

| Comportamento | Arquivo(s) | Histórico |
|---|---|---|
| Config singleton e options | `config/config.go` | `d738d60`, `eab8468`, `8fb2ccc` |
| DSN e conexão MySQL | `internal/infra/database/mysqldb/connection.go` | `8398f3e`, `7245533` |
| SQL, transação e mapeamento | `internal/infra/database/mysqldb/service.go`, `tables.go` | `8398f3e`, `91db323`, `dd57b8c` |
| Publicador SNS | `internal/infra/message/service.go` | `4c2e592`, `d8b5ad3`, `8b5a8cf` |
| Tópico, fila e assinatura | `scripts/terraform/sns.tf`, `sqs.tf`, `provider.tf` | `866dd08`, `295c681`, `d9bfc92` |
| Schema e deploy local | `scripts/k8s/*.yaml`, `scripts/sql/*.sql` | `20df627`, `ab42c68`, `da6e6f6` |
| Build e comandos locais | `Makefile`, `scripts/docker/Dockerfile`, `.env` | `bff534d`, `8dbd969`, `13c0ff2` |

## Suposições

- `.env` é local e não versionado; os valores citados vêm do arquivo presente no diretório de trabalho e do deployment k8s.
- `terraform.tfstate` versionado reflete o estado local do autor; nenhuma alteração de estado é feita por esta spec.
- A implementação do publisher pertence ao `core-sdk` desde `d8b5ad3`; este repo apenas o adapta ao contrato `Topic`.
- A spec 004 documenta observabilidade e shutdown; a spec 005 propõe mover a composição destes adapters para `cmd/main.go`.
