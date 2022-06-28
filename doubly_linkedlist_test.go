package main

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDoublyLinkedList_Insert(t *testing.T) {
	dl := DoublyLinkedList{}
	for _, dateStr := range date {
		if past, err := time.Parse(time.RFC3339, dateStr.dateString); err == nil {
			diff := time.Since(past)
			dl.Insert(diff)
		}
	}
	assert := assert2.New(t)
	assert.True(dl.Tail.Data > 0)
	dl.traverseFromHead()
	dl.traverseFromTail()
}
