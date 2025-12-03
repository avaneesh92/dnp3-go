package outstation

// Updates represents a batch of measurement updates (simplified)
type Updates struct {
	Data map[updateKey]updateValue
}

// NewUpdates creates a new Updates with the given data
func NewUpdates(data map[updateKey]updateValue) *Updates {
	return &Updates{Data: data}
}
