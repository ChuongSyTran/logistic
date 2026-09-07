package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}

func (c Config) URL() string {
	vhost := c.VHost
	if vhost == "" {
		vhost = "/"
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, trimLeadingSlash(vhost))
}

func trimLeadingSlash(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}

const DLXSuffix = ".dlx"

type Connection struct {
	cfg     Config
	mu      sync.RWMutex
	conn    *amqp.Connection
	closing bool
}

func Connect(cfg Config) (*Connection, error) {
	c := &Connection{cfg: cfg}
	if err := c.dial(); err != nil {
		return nil, err
	}
	go c.watch()
	return c, nil
}

func (c *Connection) dial() error {
	conn, err := amqp.DialConfig(c.cfg.URL(), amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})
	if err != nil {
		return fmt.Errorf("mq: dial %s:%s failed: %w", c.cfg.Host, c.cfg.Port, err)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return nil
}

func (c *Connection) watch() {
	for {
		c.mu.RLock()
		conn := c.conn
		closing := c.closing
		c.mu.RUnlock()

		if closing || conn == nil {
			return
		}

		errCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		amqpErr := <-errCh

		c.mu.RLock()
		closing = c.closing
		c.mu.RUnlock()
		if closing {
			return
		}

		log.Printf("[mq] connection lost: %v - reconnecting...", amqpErr)
		for attempt := 1; ; attempt++ {
			time.Sleep(backoff(attempt))
			if err := c.dial(); err != nil {
				log.Printf("[mq] reconnect attempt %d failed: %v", attempt, err)
				continue
			}
			log.Printf("[mq] reconnected after %d attempt(s)", attempt)
			break
		}
	}
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * time.Second
	if d > 30*time.Second {
		return 30 * time.Second
	}
	return d
}

func (c *Connection) channel() (*amqp.Channel, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil || conn.IsClosed() {
		return nil, fmt.Errorf("mq: connection is not available")
	}
	return conn.Channel()
}

func (c *Connection) Close() error {
	c.mu.Lock()
	c.closing = true
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

func (c *Connection) DeclareTopology(exchange string) error {
	ch, err := c.channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(exchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return fmt.Errorf("mq: declare exchange %s: %w", exchange, err)
	}
	if err := ch.ExchangeDeclare(exchange+DLXSuffix, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return fmt.Errorf("mq: declare dlx %s: %w", exchange+DLXSuffix, err)
	}
	return nil
}

type Publisher struct {
	conn     *Connection
	exchange string
	source   string
	mu       sync.Mutex
	ch       *amqp.Channel
}

func NewPublisher(conn *Connection, exchange, source string) (*Publisher, error) {
	if err := conn.DeclareTopology(exchange); err != nil {
		return nil, err
	}
	p := &Publisher{conn: conn, exchange: exchange, source: source}
	if err := p.refreshChannel(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Publisher) refreshChannel() error {
	ch, err := p.conn.channel()
	if err != nil {
		return err
	}

	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return fmt.Errorf("mq: enable confirm mode: %w", err)
	}
	p.ch = ch
	return nil
}

func (p *Publisher) Publish(ctx context.Context, routingKey, eventID string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mq: marshal payload for %s: %w", routingKey, err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch == nil || p.ch.IsClosed() {
		if err := p.refreshChannel(); err != nil {
			return err
		}
	}

	confirm, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    eventID,
		Timestamp:    time.Now().UTC(),
		AppId:        p.source,
		Type:         routingKey,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("mq: publish %s: %w", routingKey, err)
	}

	ackCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ok, err := confirm.WaitContext(ackCtx)
	if err != nil {
		return fmt.Errorf("mq: waiting confirm for %s: %w", routingKey, err)
	}
	if !ok {
		return fmt.Errorf("mq: broker nacked message %s (routing key %s)", eventID, routingKey)
	}
	return nil
}

func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch != nil && !p.ch.IsClosed() {
		return p.ch.Close()
	}
	return nil
}

type Delivery struct {
	RoutingKey  string
	MessageID   string
	Body        []byte
	Redelivered bool
}

type Handler func(ctx context.Context, d Delivery) error

type ConsumerConfig struct {
	Exchange    string
	Queue       string
	BindingKeys []string

	Prefetch int
}

type Consumer struct {
	conn *Connection
	cfg  ConsumerConfig
	ch   *amqp.Channel
}

func NewConsumer(conn *Connection, cfg ConsumerConfig) (*Consumer, error) {
	if cfg.Prefetch <= 0 {
		cfg.Prefetch = 20
	}
	if err := conn.DeclareTopology(cfg.Exchange); err != nil {
		return nil, err
	}

	ch, err := conn.channel()
	if err != nil {
		return nil, err
	}

	dlqName := cfg.Queue + ".dlq"

	if _, err := ch.QueueDeclare(cfg.Queue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    cfg.Exchange + DLXSuffix,
		"x-dead-letter-routing-key": dlqName,
	}); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("mq: declare queue %s: %w", cfg.Queue, err)
	}

	if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("mq: declare dlq %s: %w", dlqName, err)
	}
	if err := ch.QueueBind(dlqName, dlqName, cfg.Exchange+DLXSuffix, false, nil); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("mq: bind dlq %s: %w", dlqName, err)
	}

	for _, key := range cfg.BindingKeys {
		if err := ch.QueueBind(cfg.Queue, key, cfg.Exchange, false, nil); err != nil {
			_ = ch.Close()
			return nil, fmt.Errorf("mq: bind %s -> %s: %w", cfg.Queue, key, err)
		}
	}

	if err := ch.Qos(cfg.Prefetch, 0, false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("mq: set qos: %w", err)
	}

	return &Consumer{conn: conn, cfg: cfg, ch: ch}, nil
}

func (c *Consumer) Start(ctx context.Context, handler Handler) error {
	deliveries, err := c.ch.Consume(c.cfg.Queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("mq: consume %s: %w", c.cfg.Queue, err)
	}

	log.Printf("[mq] consuming queue=%s bindings=%v prefetch=%d", c.cfg.Queue, c.cfg.BindingKeys, c.cfg.Prefetch)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("mq: delivery channel closed for queue %s", c.cfg.Queue)
			}

			err := handler(ctx, Delivery{
				RoutingKey:  msg.RoutingKey,
				MessageID:   msg.MessageId,
				Body:        msg.Body,
				Redelivered: msg.Redelivered,
			})

			if err == nil {
				if ackErr := msg.Ack(false); ackErr != nil {
					log.Printf("[mq] ack failed for %s: %v", msg.MessageId, ackErr)
				}
				continue
			}

			requeue := !msg.Redelivered
			log.Printf("[mq] handler failed (routing=%s id=%s requeue=%v): %v",
				msg.RoutingKey, msg.MessageId, requeue, err)
			if nackErr := msg.Nack(false, requeue); nackErr != nil {
				log.Printf("[mq] nack failed for %s: %v", msg.MessageId, nackErr)
			}
		}
	}
}

func (c *Consumer) Close() error {
	if c.ch != nil && !c.ch.IsClosed() {
		return c.ch.Close()
	}
	return nil
}