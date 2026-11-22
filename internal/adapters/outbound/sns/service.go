package sns

import (
	"context"
	"time"

	accounts "github.com/clodoaldomarques/balances-api/internal/core/accounts/domain"
	"github.com/clodoaldomarques/balances-api/internal/core/accounts/ports"
	coresns "github.com/clodoaldomarques/core-sdk/pkg/aws/sns"
	"github.com/clodoaldomarques/core-sdk/pkg/zap/logger"
	"github.com/google/uuid"
)

var (
	_ ports.AccountEventPublisher = (*Publisher)(nil)
	_ ports.HealthChecker         = (*Publisher)(nil)
)

// Publisher wraps the SNS client supplied by the composition root. AWS SDK
// clients are safe for concurrent use.
type Publisher struct {
	p *coresns.Publisher
}

func NewPublisher(ctx context.Context, cfg coresns.Config) *Publisher {
	return &Publisher{p: coresns.NewPublisher(ctx, cfg)}
}

func (p Publisher) Emit(ctx context.Context, evt accounts.Event) error {
	e := coresns.Event{
		EventID:   uuid.New(),
		EventType: evt.EventType(),
		EventData: evt.EventData(),
		EventDate: time.Now(),
	}
	return p.p.Emit(ctx, e)
}

func (p Publisher) Check(ctx context.Context) error {
	return p.p.Health()
}

func (p Publisher) Close(ctx context.Context) {
	logger.Info(ctx, "ending topic connection", logger.Fields{})
}
