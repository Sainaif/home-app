package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventType represents different event types
type EventType string

const (
	EventBillCreated        EventType = "bill.created"
	EventConsumptionCreated EventType = "consumption.created"
	EventPaymentCreated     EventType = "payment.created"
	EventChoreUpdated       EventType = "chore.updated"
	EventLoanCreated        EventType = "loan.created"
	EventLoanPaymentCreated EventType = "loan.payment.created"
	EventLoanDeleted        EventType = "loan.deleted"
	EventBalanceUpdated     EventType = "balance.updated"
	EventSupplyItemAdded    EventType = "supply.item.added"
	EventSupplyItemBought   EventType = "supply.item.bought"
	EventSupplyBudgetGrew   EventType = "supply.budget.contributed"
	EventSupplyBudgetLow    EventType = "supply.budget.low"
	EventPermissionsUpdated EventType = "permissions.updated"
)

// Event represents a server-sent event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventService manages SSE connections and broadcasts events
type EventService struct {
	mu          sync.RWMutex
	subscribers map[string]chan Event // userID -> channel
}

func NewEventService() *EventService {
	return &EventService{
		subscribers: make(map[string]chan Event),
	}
}

// Subscribe creates a new SSE subscription for a user
func (s *EventService) Subscribe(userID primitive.ObjectID) chan Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan Event, 10) // Buffered channel
	s.subscribers[userID.Hex()] = ch
	return ch
}

// Unsubscribe removes a user's SSE subscription
func (s *EventService) Unsubscribe(userID primitive.ObjectID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userIDStr := userID.Hex()
	if ch, ok := s.subscribers[userIDStr]; ok {
		close(ch)
		delete(s.subscribers, userIDStr)
	}
}

// Broadcast sends an event to all subscribed users
func (s *EventService) Broadcast(eventType EventType, data map[string]interface{}) {
	event := Event{
		ID:        primitive.NewObjectID().Hex(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel buffer full, skip this subscriber
		}
	}
}

// BroadcastToUser sends an event to a specific user
func (s *EventService) BroadcastToUser(userID primitive.ObjectID, eventType EventType, data map[string]interface{}) {
	event := Event{
		ID:        primitive.NewObjectID().Hex(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	userIDStr := userID.Hex()
	if ch, ok := s.subscribers[userIDStr]; ok {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel buffer full, skip
		}
	}
}

// FormatSSE formats an event as Server-Sent Event protocol
func (e *Event) FormatSSE() string {
	data, _ := json.Marshal(e)
	return fmt.Sprintf("id: %s\nevent: %s\ndata: %s\n\n", e.ID, e.Type, string(data))
}

// SubscriberCount returns the number of active subscribers
func (s *EventService) SubscriberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}

// BroadcastToUserIDs sends an event to specific user IDs
func (s *EventService) BroadcastToUserIDs(userIDs []primitive.ObjectID, eventType EventType, data map[string]interface{}) {
	event := Event{
		ID:        primitive.NewObjectID().Hex(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, userID := range userIDs {
		userIDStr := userID.Hex()
		if ch, ok := s.subscribers[userIDStr]; ok {
			select {
			case ch <- event:
				// Event sent successfully
			default:
				// Channel buffer full, skip
			}
		}
	}
}