//go:build m2harness

package result

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateOrderService_TableDrivenCoreScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setup          func(*m2Fixture)
		req            *CreateOrderRequest
		expectErr      bool
		expectStatus   string
		expectOrderCnt int
	}{
		{
			name: "success path should commit and publish",
			setup: func(f *m2Fixture) {
				f.pricing.total = 320
			},
			req:       mustValidRequest("req-success"),
			expectErr: false, expectStatus: "CREATED", expectOrderCnt: 1,
		},
		{
			name: "invalid request should fail fast",
			setup: func(f *m2Fixture) {
				f.pricing.total = 1
			},
			req: &CreateOrderRequest{
				RequestID: "",
				UserID:    "u1",
				Items:     []OrderItem{{SKU: "sku-1", Quantity: 1, UnitPrice: 10}},
				Currency:  "CNY",
			},
			expectErr: true, expectStatus: "", expectOrderCnt: 0,
		},
		{
			name: "pricing failure should not begin tx",
			setup: func(f *m2Fixture) {
				f.pricing.err = errors.New("pricing down")
			},
			req:            mustValidRequest("req-pricing-fail"),
			expectErr:      true,
			expectStatus:   "",
			expectOrderCnt: 0,
		},
		{
			name: "inventory failure should not begin tx",
			setup: func(f *m2Fixture) {
				f.pricing.total = 120
				f.inventory.err = errors.New("inventory busy")
			},
			req:            mustValidRequest("req-inv-fail"),
			expectErr:      true,
			expectStatus:   "",
			expectOrderCnt: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture()
			tt.setup(f)
			resp, err := f.svc.HandleCreateOrder(context.Background(), tt.req)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectStatus, resp.Status)
			assert.Len(t, f.repo.orders, tt.expectOrderCnt)
			assert.Equal(t, 1, f.producer.publishCalls)
		})
	}
}

func TestCreateOrderService_TxFailureShouldRollback(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.pricing.total = 88
	f.repo.failCreateOrder = true

	_, err := f.svc.HandleCreateOrder(context.Background(), mustValidRequest("req-tx-fail"))
	assert.Error(t, err)
	assert.Equal(t, 1, f.txManager.rollbackCalls)
	assert.Equal(t, 0, f.txManager.commitCalls)
	assert.Equal(t, 0, f.producer.publishCalls)
}

func TestCreateOrderService_KafkaFailureShouldKeepConsistencyAndRetryMark(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.pricing.total = 500
	f.producer.err = errors.New("kafka unavailable")

	_, err := f.svc.HandleCreateOrder(context.Background(), mustValidRequest("req-kafka-fail"))
	assert.Error(t, err)
	assert.Len(t, f.repo.orders, 1, "order should be persisted before publish")
	assert.Len(t, f.repo.outboxes, 1, "outbox should be persisted in tx")
	assert.Equal(t, 1, f.repo.markRetryCalls, "outbox retry state should be marked")
	assert.GreaterOrEqual(t, f.cache.setCalls, 1, "redis retry marker should be written")
}

func TestCreateOrderService_IdempotencyShouldNotDuplicateOrder(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.pricing.total = 111
	req := mustValidRequest("req-idempotent")

	resp1, err1 := f.svc.HandleCreateOrder(context.Background(), req)
	assert.NoError(t, err1)
	resp2, err2 := f.svc.HandleCreateOrder(context.Background(), req)
	assert.NoError(t, err2)

	assert.NotNil(t, resp1)
	assert.NotNil(t, resp2)
	assert.Equal(t, resp1.OrderID, resp2.OrderID)
	assert.Len(t, f.repo.orders, 1, "same request_id must not create duplicate orders")
}

func TestCreateOrderService_NoRemoteCallDuringTx(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.pricing.total = 66

	_, err := f.svc.HandleCreateOrder(context.Background(), mustValidRequest("req-no-rpc-in-tx"))
	assert.NoError(t, err)
	assert.False(t, f.pricing.calledInTx)
	assert.False(t, f.inventory.calledInTx)
}

func mustValidRequest(requestID string) *CreateOrderRequest {
	return &CreateOrderRequest{
		RequestID: requestID,
		UserID:    "u-100",
		Items: []OrderItem{
			{SKU: "sku-a", Quantity: 2, UnitPrice: 30},
			{SKU: "sku-b", Quantity: 1, UnitPrice: 50},
		},
		Currency: "CNY",
	}
}

type m2Fixture struct {
	repo      *fakeRepo
	txManager *fakeTxManager
	pricing   *fakePricing
	inventory *fakeInventory
	producer  *fakeProducer
	cache     *fakeCache
	svc       *CreateOrderService
}

func newFixture() *m2Fixture {
	txm := &fakeTxManager{}
	repo := &fakeRepo{}
	pricing := &fakePricing{txm: txm}
	inventory := &fakeInventory{txm: txm}
	producer := &fakeProducer{}
	cache := &fakeCache{locks: map[string]bool{}}

	return &m2Fixture{
		repo:      repo,
		txManager: txm,
		pricing:   pricing,
		inventory: inventory,
		producer:  producer,
		cache:     cache,
		svc:       NewCreateOrderService(repo, txm, inventory, pricing, producer, cache),
	}
}

type fakeTx struct{ manager *fakeTxManager }

func (f *fakeTx) Commit() error {
	f.manager.mu.Lock()
	defer f.manager.mu.Unlock()
	f.manager.inTx = false
	f.manager.commitCalls++
	if f.manager.commitErr != nil {
		return f.manager.commitErr
	}
	return nil
}

func (f *fakeTx) Rollback() error {
	f.manager.mu.Lock()
	defer f.manager.mu.Unlock()
	if f.manager.inTx {
		f.manager.rollbackCalls++
	}
	f.manager.inTx = false
	return nil
}

type fakeTxManager struct {
	mu            sync.Mutex
	inTx          bool
	beginCalls    int
	commitCalls   int
	rollbackCalls int
	commitErr     error
}

func (f *fakeTxManager) BeginTx(ctx context.Context) (Tx, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inTx = true
	f.beginCalls++
	return &fakeTx{manager: f}, nil
}

func (f *fakeTxManager) IsInTx() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inTx
}

type fakeRepo struct {
	mu              sync.Mutex
	orders          []*Order
	outboxes        []*OutboxEvent
	failCreateOrder bool
	markRetryCalls  int
}

func (f *fakeRepo) FindByRequestID(ctx context.Context, requestID string) (*Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, o := range f.orders {
		if o.RequestID == requestID {
			cp := *o
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) CreateOrder(ctx context.Context, tx Tx, order *Order) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failCreateOrder {
		return errors.New("create order failed")
	}
	cp := *order
	f.orders = append(f.orders, &cp)
	return nil
}

func (f *fakeRepo) CreateOutbox(ctx context.Context, tx Tx, evt *OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *evt
	f.outboxes = append(f.outboxes, &cp)
	return nil
}

func (f *fakeRepo) MarkOutboxPublished(ctx context.Context, eventID string) error { return nil }

func (f *fakeRepo) MarkOutboxRetry(ctx context.Context, eventID string, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markRetryCalls++
	return nil
}

type fakePricing struct {
	txm        *fakeTxManager
	total      int64
	err        error
	calledInTx bool
}

func (f *fakePricing) Calculate(ctx context.Context, userID string, items []OrderItem, currency string) (int64, error) {
	if f.txm != nil && f.txm.IsInTx() {
		f.calledInTx = true
	}
	if f.err != nil {
		return 0, f.err
	}
	if f.total > 0 {
		return f.total, nil
	}
	var sum int64
	for _, it := range items {
		sum += int64(it.Quantity) * it.UnitPrice
	}
	return sum, nil
}

type fakeInventory struct {
	txm        *fakeTxManager
	err        error
	calledInTx bool
}

func (f *fakeInventory) Reserve(ctx context.Context, userID string, items []OrderItem) error {
	if f.txm != nil && f.txm.IsInTx() {
		f.calledInTx = true
	}
	return f.err
}

type fakeProducer struct {
	err          error
	publishCalls int
}

func (f *fakeProducer) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	f.publishCalls++
	return f.err
}

type fakeCache struct {
	mu       sync.Mutex
	locks    map[string]bool
	setCalls int
}

func (f *fakeCache) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.locks[key] {
		return false, nil
	}
	f.locks[key] = true
	return true, nil
}

func (f *fakeCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setCalls++
	return nil
}

func (f *fakeCache) Delete(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.locks, key)
	return nil
}
