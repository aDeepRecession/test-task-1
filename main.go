package main

import (
	"fmt"
)

type Queue interface {
	Poll() (string, error)
	Offer(string)
	Size() int
	IsEmpty() bool
}

type QueueImpl struct {
	arr []string
}

func (q *QueueImpl) Offer(val string) {
	q.arr = append(q.arr, val)
}

func (q *QueueImpl) Poll() (string, error) {
	if q.IsEmpty() {
		return "", fmt.Errorf("queue is empty")
	}
	headVal := q.arr[0]
	q.arr = q.arr[1:]
	return headVal, nil
}

func (q *QueueImpl) Size() int {
	return len(q.arr)
}

func (q *QueueImpl) IsEmpty() bool {
	return len(q.arr) == 0
}

var queueMap = map[string]QueueImpl{}
