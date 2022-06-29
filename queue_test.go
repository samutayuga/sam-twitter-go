package main

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDurationQueue_Enqueue(t *testing.T) {
	aQueue := DurationQueue{}
	for _, dateStr := range date {
		if past, err := time.Parse(time.RFC3339, dateStr.dateString); err == nil {
			diff := time.Since(past)
			aQueue.Enqueue(diff)
		}
	}
	assert := assert2.New(t)
	a := aQueue.Dequeue()
	assert.True(a.Seconds() > 0)
	a = aQueue.Dequeue()
	assert.True(a.Seconds() > 0)
	a = aQueue.Dequeue()
	assert.True(a.Seconds() > 0)
	a = aQueue.Dequeue()
	assert.True(a.Seconds() > 0)
	//ops
	a = aQueue.Dequeue()
	assert.True(a.Seconds() == 0)

}

func TestDateQueue_Dequeue(t *testing.T) {

}
