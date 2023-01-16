package dlm

import "time"

//go:generate mockgen -package mock -destination=mock/dlm.go . DLM

// DLM Distributed Lock Manager interface
type DLM interface {
	// Lock locks a given name until the expiry duration has passed or Unlock is called
	Lock(name string, expiry time.Duration) error
	// Unlock removes a lock from a given name
	Unlock(name string) (bool, error)
}
