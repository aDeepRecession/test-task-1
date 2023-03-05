package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	t.Run("offer offer poll poll", func(t *testing.T) {
		q := QueueImpl{}
		val1 := "1"
		val2 := "2"

		q.Offer(val1)
		q.Offer(val2)
		poll1, err1 := q.Poll()
		poll2, err2 := q.Poll()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, val1, poll1)
		assert.Equal(t, val2, poll2)
	})

	t.Run("size", func(t *testing.T) {
		q := QueueImpl{}

		q.Offer("")
		q.Offer("")

		expected := 2
		got := q.Size()
		assert.Equal(t, got, expected)
	})

	t.Run("is empty", func(t *testing.T) {
		q := QueueImpl{}

		q.Offer("")
		q.Poll()

		expected := true
		got := q.IsEmpty()
		assert.Equal(t, got, expected)
	})

	t.Run("Poll on empty should return an error", func(t *testing.T) {
		q := QueueImpl{}

		val, err := q.Poll()

		assert.Error(t, err, "queue is empty")
		assert.Equal(t, val, "")
	})
}
