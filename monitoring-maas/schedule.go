package maas

import (
	"time"
)

const defaultFrequency = 10 * time.Second

type Schedule struct {
	frequency time.Duration
	timeout   time.Duration
	isEnabled bool
}

func NewSchedule(options ...func(*Schedule)) *Schedule {
	s := &Schedule{
		frequency: defaultFrequency,
		timeout:   time.Second,
		isEnabled: true,
	}

	s.apply(options)

	return s
}

func WithFrequency(f time.Duration) func(*Schedule) {
	return func(s *Schedule) {
		s.frequency = f
	}
}

func WithTimeout(t time.Duration) func(*Schedule) {
	return func(s *Schedule) {
		s.timeout = t
	}
}

func Disabled() func(*Schedule) {
	return func(s *Schedule) {
		s.isEnabled = false
	}
}

func (s *Schedule) apply(options []func(*Schedule)) {
	for _, option := range options {
		option(s)
	}
}
