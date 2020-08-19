package main

import (
	"fmt"
	"time"
)

type Timing struct {
	Label    string
	Duration time.Duration
}

func (t Timing) String() string {
	return fmt.Sprintf("%s: %s", t.Label, t.Duration)
}

type Timings []Timing

func (t Timings) Len() int {
	return len(t)
}

func (t Timings) Less(i, j int) bool {
	return t[i].Duration < t[j].Duration
}

func (t Timings) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
