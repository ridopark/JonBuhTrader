package backtester

import (
	"time"

	"github.com/ridopark/JonBuhTrader/pkg/strategy"
)

// Config represents the backtester configuration
type Config struct {
	StartDate      time.Time `yaml:"start_date"`
	EndDate        time.Time `yaml:"end_date"`
	InitialCapital float64   `yaml:"initial_capital"`
	Commission     float64   `yaml:"commission"`
	Symbols        []string  `yaml:"symbols"`
	Timeframe      string    `yaml:"timeframe"`
}

// Event represents different types of events in the backtester
type Event interface {
	GetTimestamp() time.Time
	GetType() EventType
}

// EventType represents the type of event
type EventType string

const (
	EventTypeBar   EventType = "BAR"
	EventTypeOrder EventType = "ORDER"
	EventTypeFill  EventType = "FILL"
)

// BarEvent represents a new bar of market data
type BarEvent struct {
	Bar strategy.BarData
}

func (e BarEvent) GetTimestamp() time.Time {
	return e.Bar.Timestamp
}

func (e BarEvent) GetType() EventType {
	return EventTypeBar
}

// OrderEvent represents an order to be executed
type OrderEvent struct {
	Order strategy.Order
}

func (e OrderEvent) GetTimestamp() time.Time {
	return e.Order.Timestamp
}

func (e OrderEvent) GetType() EventType {
	return EventTypeOrder
}

// FillEvent represents a completed trade
type FillEvent struct {
	Trade strategy.TradeEvent
}

func (e FillEvent) GetTimestamp() time.Time {
	return e.Trade.Timestamp
}

func (e FillEvent) GetType() EventType {
	return EventTypeFill
}

// EventQueue manages the event queue for the backtester
type EventQueue struct {
	events []Event
}

// NewEventQueue creates a new event queue
func NewEventQueue() *EventQueue {
	return &EventQueue{
		events: make([]Event, 0),
	}
}

// Push adds an event to the queue
func (eq *EventQueue) Push(event Event) {
	eq.events = append(eq.events, event)
}

// Pop removes and returns the next event from the queue
func (eq *EventQueue) Pop() Event {
	if len(eq.events) == 0 {
		return nil
	}

	event := eq.events[0]
	eq.events = eq.events[1:]
	return event
}

// IsEmpty returns true if the queue is empty
func (eq *EventQueue) IsEmpty() bool {
	return len(eq.events) == 0
}

// Len returns the number of events in the queue
func (eq *EventQueue) Len() int {
	return len(eq.events)
}
