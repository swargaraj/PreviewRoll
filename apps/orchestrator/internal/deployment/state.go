package deployment

import (
	"fmt"
)

// State represents the deployment state
type State string

const (
	StateQueued      State = "queued"
	StateAssigned    State = "assigned"
	StateBuilding    State = "building"
	StateRunning     State = "running"
	StateInterrupted State = "interrupted"
	StateFailed      State = "failed"
	StateStopped     State = "stopped"
	StateDestroyed   State = "destroyed"
)

// ValidTransitions defines the valid state transitions
var ValidTransitions = map[State][]State{
	StateQueued:      {StateAssigned, StateStopped},
	StateAssigned:    {StateBuilding, StateQueued, StateFailed},
	StateBuilding:    {StateRunning, StateFailed, StateInterrupted, StateStopped},
	StateRunning:     {StateFailed, StateStopped, StateDestroyed},
	StateInterrupted: {StateQueued, StateFailed},
	StateFailed:      {StateQueued, StateDestroyed},
	StateStopped:     {StateDestroyed},
	StateDestroyed:   {},
}

// Transition represents a state transition
type Transition struct {
	From      State
	To        State
	Reason    string
	Timestamp int64
}

// CanTransition checks if a transition is valid
func CanTransition(from, to State) bool {
	allowed, exists := ValidTransitions[from]
	if !exists {
		return false
	}

	for _, valid := range allowed {
		if valid == to {
			return true
		}
	}

	return false
}

// ValidateTransition validates and returns error for invalid transitions
func ValidateTransition(from, to State) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid transition from %s to %s", from, to)
	}
	return nil
}

// IsTerminal checks if a state is terminal (no outgoing transitions)
func IsTerminal(state State) bool {
	transitions, exists := ValidTransitions[state]
	return exists && len(transitions) == 0
}

// IsRetryable checks if a failed deployment can be retried
func IsRetryable(state State) bool {
	return state == StateFailed || state == StateInterrupted
}

// String returns the string representation of the state
func (s State) String() string {
	return string(s)
}
