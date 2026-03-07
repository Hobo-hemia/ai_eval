package input

import "context"

// TODO: 根据真实题目替换成正式依赖接口。
type OrderRepository interface {
	Save(ctx context.Context, orderID string) error
}

type EventProducer interface {
	Send(ctx context.Context, topic string, payload []byte) error
}
