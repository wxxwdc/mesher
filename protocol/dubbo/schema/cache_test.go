package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFreshTicker(t *testing.T) {
	r1 := newRefresher(2 * time.Second)
	r1.Run()

	a1, a2 := 0, 0
	r1.Add(Job{Fn: func() { a1++ }})
	time.Sleep(1 * time.Second)
	r1.Add(Job{Fn: func() { a2++ }})

	select {
	case <-time.After(5 * time.Second):
	}
	assert.Equal(t, a1, 2)
	assert.Equal(t, a2, 2)
}
