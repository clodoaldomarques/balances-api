# Balances API

> Go-based backend service for balance management, integrating asynchronous messaging, AWS-compatible infrastructure and a cloud-native runtime.

## Overview

Balances API is a Go backend service responsible for balance-related operations within a distributed financial services architecture.

The project was designed to explore the implementation of a cloud-native backend service integrating synchronous APIs with asynchronous messaging and supporting infrastructure.

The application is designed to run in a Kubernetes environment and uses AWS-compatible services for asynchronous communication.

The project also includes local infrastructure resources for development using Minikube and LocalStack.

## Architecture

The application follows a layered and modular backend structure, separating application initialization, configuration, domain logic and infrastructure concerns.

```text
                         ┌──────────────────────┐
                         │     Client / API     │
                         └──────────┬───────────┘
                                    │
                                  HTTP
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │    Balances API      │
                         │                      │
                         │        Go            │
                         └──────────┬───────────┘
                                    │
                     ┌──────────────┼──────────────┐
                     │              │              │
                     ▼              ▼              ▼
                ┌─────────┐   ┌───────────┐   ┌──────────┐
                │  MySQL  │   │    SNS    │   │   SQS    │
                │         │   │           │   │          │
                │Balances │   │  Events   │   │ Messages │
                └─────────┘   └─────┬─────┘   └────┬─────┘
                                    │              │
                                    └──────┬───────┘
                                           │
                                           ▼
                                  Asynchronous Workers
```

The application is part of a broader distributed-system environment where synchronous API operations can be combined with asynchronous event processing.

## Main Responsibilities

The service provides the backend foundation for balance-related operations.

Its responsibilities include:

- Balance management
- Persistence of balance-related data
- API-based access to balance operations
- Integration with asynchronous messaging
- Publication of application events
- Consumption of asynchronous messages
- Integration with AWS-compatible services
- Database migration management

## Event-Driven Architecture

One of the main characteristics of the project is the integration between synchronous HTTP operations and asynchronous messaging.

```text
                  HTTP Request
                       │
                       ▼
              ┌─────────────────┐
              │   Balances API  │
              └────────┬────────┘
                       │
                       │
              ┌────────▼────────┐
              │    Business     │
              │     Logic       │
              └────────┬────────┘
                       │
                ┌──────┴───────┐
                │              │
                ▼              ▼
             MySQL            SNS
                                │
                                │
                                ▼
                               SQS
                                │
                                ▼
                        Async Processing
```

This architecture decouples producers from consumers and allows asynchronous processing to evolve independently from the API layer.

## AWS Integration

The application integrates with AWS services through the AWS SDK for Go.

The messaging architecture uses:

- Amazon SNS
- Amazon SQS

SNS is used for event publication while SQS provides asynchronous message consumption.

For local development, these integrations can be executed against LocalStack instead of requiring a real AWS account.

## LocalStack

LocalStack provides AWS-compatible services for local development.

This allows the application to exercise AWS integrations while running entirely in a local environment.

The local environment can therefore reproduce important parts of the distributed architecture without depending on external cloud resources.

```text
                    Local Environment
                           │
                           ▼
                     ┌───────────┐
                     │ LocalStack│
                     └─────┬─────┘
                           │
              ┌────────────┼────────────┐
              │                         │
              ▼                         ▼
           Amazon SNS               Amazon SQS
```

## Database

The application uses MySQL for persistence.

Database schema evolution is managed through Flyway migrations.

This allows database changes to be versioned together with the application and applied in a controlled and reproducible way.

## Database Migrations

Migration scripts are maintained as versioned SQL files.

The migration strategy allows:

- Schema versioning
- Reproducible database initialization
- Incremental schema evolution
- Separation between application code and database migration execution

## Kubernetes

The application is designed to run in Kubernetes.

The repository includes deployment resources used to run the application and its dependencies in a local Kubernetes environment.

The application can therefore be developed using a local environment that closely resembles a cloud-native deployment model.

```text
                    Minikube
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
   Balances API     MySQL        LocalStack
                                      │
                                 ┌────┴────┐
                                 │         │
                                SNS       SQS
```

## Infrastructure as Code

The project uses Terraform to provision messaging infrastructure.

Terraform is responsible for defining AWS-compatible messaging resources used by the application.

This allows infrastructure configuration to be versioned alongside the application and reproduced consistently across environments.

For local development, the Terraform configuration can target LocalStack.

## Project Structure

```text
balances-api/
│
├── cmd/
│   └── Application entry point
│
├── config/
│   └── Application configuration
│
├── docs/
│   └── Architectural documentation
│
├── internal/
│   └── Application and domain implementation
│
├── scripts/
│   └── Infrastructure and deployment resources
│
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

### `cmd/`

Contains the application entry point and service initialization.

### `config/`

Contains application configuration and environment-related settings.

### `internal/`

Contains the application's internal implementation, keeping implementation details encapsulated within the service.

### `docs/`

Contains architectural documentation and system models.

### `scripts/`

Contains infrastructure and deployment resources used by the project.

## Local Development

### Requirements

- Go
- Docker
- Minikube
- kubectl
- Terraform
- Flyway

### Clone

```bash
git clone https://github.com/clodoaldomarques/balances-api.git
cd balances-api
```

### Install dependencies

```bash
go mod download
```

### Run tests

```bash
go test ./...
```

### Build

```bash
go build ./...
```

## Kubernetes Environment

The project can be executed in a local Kubernetes environment using Minikube.

Start Minikube:

```bash
minikube start
```

Apply the required Kubernetes resources:

```bash
kubectl apply -f scripts/k8s/
```

Verify the deployment:

```bash
kubectl get pods
```

Verify the services:

```bash
kubectl get services
```

## Messaging

The project uses asynchronous messaging to decouple application components.

The messaging flow is based on the following model:

```text
                  ┌──────────────┐
                  │ Balances API │
                  └──────┬───────┘
                         │
                         │ Publish
                         ▼
                    ┌─────────┐
                    │   SNS   │
                    └────┬────┘
                         │
                         │ Subscription
                         ▼
                    ┌─────────┐
                    │   SQS   │
                    └────┬────┘
                         │
                         │ Consume
                         ▼
                  ┌──────────────┐
                  │    Worker    │
                  └──────────────┘
```

This pattern provides asynchronous communication between services and reduces direct coupling between producers and consumers.

## Technology Stack

| Technology | Purpose |
|---|---|
| Go | Backend service |
| AWS SDK for Go v2 | AWS integration |
| Amazon SNS | Event publication |
| Amazon SQS | Asynchronous messaging |
| MySQL | Relational persistence |
| Flyway | Database migrations |
| Kubernetes | Container orchestration |
| Minikube | Local Kubernetes environment |
| Docker | Containerization |
| LocalStack | Local AWS environment |
| Terraform | Infrastructure as Code |

## Engineering Concepts

This project demonstrates practical experience with:

- Go backend development
- REST APIs
- Microservices architecture
- Event-driven architecture
- Asynchronous messaging
- Amazon SNS
- Amazon SQS
- AWS SDK for Go v2
- Kubernetes
- Minikube
- Docker
- LocalStack
- Terraform
- MySQL
- Database migrations
- Infrastructure as Code
- Distributed systems

## Design Decisions

### Why asynchronous messaging?

Balance-related operations can generate events that do not necessarily need to be processed synchronously by the API.

Using SNS and SQS allows these events to be propagated asynchronously and consumed independently.

### Why LocalStack?

LocalStack provides an AWS-compatible environment for local development.

This makes it possible to develop and test AWS integrations without requiring access to a remote AWS environment.

### Why Terraform?

Infrastructure should be reproducible and version controlled just like application code.

Terraform provides a declarative way to define the messaging infrastructure required by the application.

### Why Flyway?

Database schema changes are part of the application's lifecycle.

Flyway provides versioned migrations that allow database evolution to be tracked and reproduced consistently.

## Cloud-Native Development

The project combines application code and infrastructure concerns required by a distributed backend:

```text
                    Application
                         │
                         ▼
                       Go API
                         │
            ┌────────────┼────────────┐
            │            │            │
            ▼            ▼            ▼
          MySQL         SNS          SQS
                          │            │
                          └─────┬──────┘
                                │
                                ▼
                         Async Processing

                    Infrastructure Layer
                         │
             ┌───────────┼───────────┐
             ▼           ▼           ▼
         Kubernetes   Terraform   LocalStack
```

The result is a local environment capable of reproducing the main infrastructure dependencies of the application.

## Relationship with the Ledger Ecosystem

Balances API can be considered one of the core services in the broader Ledger project ecosystem.

```text
                         ┌───────────────────┐
                         │   Ledger Config   │
                         └───────────────────┘
                                  │
                                  │
                         ┌────────▼─────────┐
                         │  Balances API    │
                         │                  │
                         │  Balance Domain  │
                         └────────┬─────────┘
                                  │
                             Events
                                  │
                                  ▼
                         ┌───────────────────┐
                         │   Ledger Events   │
                         └────────┬──────────┘
                                  │
                                  ▼
                         ┌───────────────────┐
                         │   Ledger Worker   │
                         └───────────────────┘
```

The projects can be used together to demonstrate a distributed architecture involving APIs, event publication, asynchronous processing and shared infrastructure.

## Project Status

This project is part of my Go backend engineering portfolio and serves as a practical exploration of distributed systems, event-driven architecture, AWS integrations and cloud-native application development.

The project is intended primarily as an engineering and architecture study rather than a production-ready financial platform.

## Author

**Clodoaldo Marques**

Backend Software Engineer focused on Go, Microservices, Distributed Systems and Cloud-Native architectures.

- GitHub: https://github.com/clodoaldomarques
- LinkedIn: https://www.linkedin.com/in/clodoaldomarques/