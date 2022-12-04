package puched

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
)

// State is a type that represents a state of CircuitBreaker.
type State int

// These constants are states of Green Service.
const (
	StateMidful = iota
	StateAnnoy
	StateRage
)

// Handler is .......
type Handler struct {
	EmoState    State
	Counter     int
	MidfulLimit int
	AnnoyLimit  int
	expiry      time.Time

	mutex sync.Mutex
}

var zero time.Time

// New PuchedHandler
func New() *Handler {
	p := Handler{
		EmoState:    StateMidful,
		Counter:     0,
		MidfulLimit: 200,
		AnnoyLimit:  100,
		expiry:      zero,
	}
	return &p
}

// Hit ...
func (p *Handler) Hit(ctx context.Context) (int, State) {
	tr := otel.Tracer("puched")
	p.mutex.Lock()
	defer p.mutex.Unlock()
	_, span := tr.Start(ctx, "Hit")
	defer span.End()

	now := time.Now()
	p.Counter++
	switch p.EmoState {
	case StateMidful:
		{
			if p.Counter >= p.MidfulLimit {
				p.Counter = 0
				p.EmoState = StateAnnoy
				p.expiry = now.Add(10 * time.Second)
			}
		}
	case StateAnnoy:
		{
			if p.expiry.Before(now) && p.Counter <= p.AnnoyLimit {
				p.Counter = 1
				p.EmoState = StateMidful
				p.expiry = zero
			}

			if p.Counter >= p.AnnoyLimit {
				p.Counter = 0
				p.EmoState = StateRage
				p.expiry = now.Add(5 * time.Second)
			}
		}
	case StateRage:
		{
			if p.expiry.Before(now) && p.Counter == 0 {
				p.Counter = 1
				p.EmoState = StateMidful
				p.expiry = zero
			} else {
				p.Counter = 0
				p.expiry = now.Add(10 * time.Second)
			}
		}
	}

	return p.Counter, p.EmoState
}
