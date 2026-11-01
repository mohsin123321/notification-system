package rabbit

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// DeclareTopology declares all exchanges and queues required by the notification system.
func DeclareTopology(ch *amqp.Channel) error {
	// --- Exchanges ---

	// Primary exchange for dispatching notifications to workers.
	if err := ch.ExchangeDeclare("notifications", "topic", true, false, false, false, nil); err != nil {
		return err
	}

	// Exchange for worker result events (status updates).
	if err := ch.ExchangeDeclare("notification.events", "topic", true, false, false, false, nil); err != nil {
		return err
	}

	// amq.topic is a built-in RabbitMQ exchange used by MQTT; no declaration needed.

	// --- DLX / Retry Exchanges ---
	// We use the default exchange for direct routing to retry queues,
	// but we need a named DLX to route from main queues to DLQ on final failure.
	if err := ch.ExchangeDeclare("notifications.dlx", "direct", true, false, false, false, nil); err != nil {
		return err
	}

	// --- Main Queues ---
	mainArgs := func(retryQueue string) amqp.Table {
		return amqp.Table{
			"x-dead-letter-exchange":    "",     // default exchange (direct to named queue)
			"x-dead-letter-routing-key": retryQueue,
		}
	}

	if _, err := ch.QueueDeclare("q.email.high", true, false, false, false, mainArgs("q.email.retry")); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("q.email.low", true, false, false, false, mainArgs("q.email.retry")); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("q.sms", true, false, false, false, mainArgs("q.sms.retry")); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("q.inapp", true, false, false, false, mainArgs("q.inapp.retry")); err != nil {
		return err
	}

	// --- Retry Queues (TTL → DLX back to notifications exchange) ---
	retryArgs := func(origQueue string) amqp.Table {
		return amqp.Table{
			"x-message-ttl":             int32(5000), // 5 second TTL before re-routing
			"x-dead-letter-exchange":    "notifications",
			"x-dead-letter-routing-key": origQueue,
		}
	}

	if _, err := ch.QueueDeclare("q.email.retry", true, false, false, false, retryArgs("notify.email.high")); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("q.sms.retry", true, false, false, false, retryArgs("notify.sms.high")); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("q.inapp.retry", true, false, false, false, retryArgs("notify.inapp.high")); err != nil {
		return err
	}

	// --- DLQs ---
	if _, err := ch.QueueDeclare("q.email.dlq", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("q.sms.dlq", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare("q.inapp.dlq", true, false, false, false, nil); err != nil {
		return err
	}

	// --- Bindings: notifications exchange → main queues ---
	bindings := []struct{ queue, key string }{
		{"q.email.high", "notify.email.high"},
		{"q.email.low", "notify.email.low"},
		{"q.sms", "notify.sms.high"},
		{"q.sms", "notify.sms.low"},
		{"q.inapp", "notify.inapp.high"},
		{"q.inapp", "notify.inapp.low"},
	}
	for _, b := range bindings {
		if err := ch.QueueBind(b.queue, b.key, "notifications", false, nil); err != nil {
			return err
		}
	}

	return nil
}
