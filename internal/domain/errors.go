package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrContextOverflow = errors.New("context overflow")
	ErrAuthRotation    = errors.New("auth rotation required")
	ErrPoisonPill      = errors.New("poison pill")
)
