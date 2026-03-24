package model

type CollectorState string

const (
	CollectorStateReady   CollectorState = "ready"
	CollectorStateMissing CollectorState = "missing"
	CollectorStateTimeout CollectorState = "timeout"
	CollectorStateError   CollectorState = "error"
)

type CollectorStatus struct {
	Source  Source
	Label   string
	State   CollectorState
	Details string
}
