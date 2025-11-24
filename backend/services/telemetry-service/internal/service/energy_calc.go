package service

// CalculateDeltaEnergy computes incremental energy based on previous and current meter values.
func CalculateDeltaEnergy(prev, current float64) float64 {
	if current < prev {
		return 0
	}
	return current - prev
}

