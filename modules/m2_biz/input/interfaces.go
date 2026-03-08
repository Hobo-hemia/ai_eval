package result

import (
	"context"
	"time"
)

type OrderItem struct {
	SKU       string
	Quantity  int32
	UnitPrice int64
}

type CreateOrderRequest struct {
	RequestID string
	UserID    string
	Items     []OrderItem
	Currency  string
}

type CreateOrderResponse struct {
	OrderID     string
	TotalAmount int64
	Status      string
}

type Order struct {
	OrderID     string
	RequestID   string
	UserID      string
	TotalAmount int64
	Status      string
	CreatedAt   int64
}

type OutboxEvent struct {
	EventID string
	OrderID string
	Topic   string
	Key     string
	Payload []byte
}

type Tx interface {
	Commit() error
	Rollback() error
}

type TxManager interface {
	BeginTx(ctx context.Context) (Tx, error)
}

type OrderRepository interface {
	FindByRequestID(ctx context.Context, requestID string) (*Order, error)
	CreateOrder(ctx context.Context, tx Tx, order *Order) error
	CreateOutbox(ctx context.Context, tx Tx, evt *OutboxEvent) error
	MarkOutboxPublished(ctx context.Context, eventID string) error
	MarkOutboxRetry(ctx context.Context, eventID string, reason string) error
}

type PricingClient interface {
	Calculate(ctx context.Context, userID string, items []OrderItem, currency string) (int64, error)
}

type InventoryClient interface {
	Reserve(ctx context.Context, userID string, items []OrderItem) error
}

type KafkaProducer interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}

type RedisCache interface {
	SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
