package puched

import (
	"fmt"
	"sync"
	"time"
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
		MidfulLimit: 20,
		AnnoyLimit:  10,
		expiry:      zero,
	}
	return &p
}

// Hit ...
func (p *Handler) Hit() (int, State) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
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
				p.expiry = now.Add(30 * time.Second)
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
				p.expiry = now.Add(30 * time.Second)
			}
		}
	}

	fmt.Println("counter", p.Counter, "state", p.EmoState)
	return p.Counter, p.EmoState
}
