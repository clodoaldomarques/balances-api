package message

import (
	"context"
	"time"

	"github.com/clodoaldomarques/balances-api/config"
	"github.com/clodoaldomarques/balances-api/internal/domain/accounts"
	"github.com/clodoaldomarques/core-sdk/pkg/logger"
	"github.com/clodoaldomarques/core-sdk/pkg/sns"
	"github.com/google/uuid"
)

type Topic struct {
	p sns.Publisher
}

func NewTopic(ctx context.Context) *Topic {
	return &Topic{
		p: *sns.NewPublisher(ctx, config.New()),
	}
}

func (t Topic) Emit(ctx context.Context, evt accounts.Event) error {
	e := sns.Event{
		EventID:   uuid.New(),
		EventType: evt.EventType(),
		EventData: evt.EventData(),
		EventDate: time.Now(),
	}
	return t.p.Emit(ctx, e)
}

func (t Topic) Close(ctx context.Context) {
	logger.Info(ctx, "ending topic connection", logger.Fields{})
}
