package common

import "sync"

// Message represents an internal message for the queue.
type Message struct {
	ID      string
	Topic   string
	Payload interface{}
}

// MessageQueue is a simple thread-safe message queue.
type MessageQueue struct {
	mu       sync.Mutex
	messages []Message
	cond     *sync.Cond
	closed   bool
}

// NewMessageQueue creates a new message queue.
func NewMessageQueue() *MessageQueue {
	mq := &MessageQueue{}
	mq.cond = sync.NewCond(&mq.mu)
	return mq
}

// Enqueue adds a message to the queue.
func (mq *MessageQueue) Enqueue(msg Message) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	if mq.closed {
		return ErrClosed
	}
	mq.messages = append(mq.messages, msg)
	mq.cond.Signal()
	return nil
}

// Dequeue removes and returns a message (blocks if empty).
func (mq *MessageQueue) Dequeue() (Message, error) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	for len(mq.messages) == 0 {
		if mq.closed {
			return Message{}, ErrClosed
		}
		mq.cond.Wait()
	}
	msg := mq.messages[0]
	mq.messages = mq.messages[1:]
	return msg, nil
}

// Len returns the current queue length.
func (mq *MessageQueue) Len() int {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return len(mq.messages)
}

// Close closes the queue.
func (mq *MessageQueue) Close() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.closed = true
	mq.cond.Broadcast()
}

// DequeueWithTimeout attempts to dequeue with a bounded wait (non-blocking if empty).
func (mq *MessageQueue) DequeueWithTimeout() (Message, bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	if len(mq.messages) == 0 {
		return Message{}, false
	}
	msg := mq.messages[0]
	mq.messages = mq.messages[1:]
	return msg, true
}
