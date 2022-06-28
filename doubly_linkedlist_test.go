package main

import (
	"fmt"
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
	fmt.Println("Traverse before delete")
	dl.traverseFromHead()
	dl.traverseFromTail()

	//1. get current tail
	currentTail := dl.Tail

	tailDeleted := dl.Delete()

	assert.Equal(currentTail, tailDeleted)
	fmt.Println("Traverse after delete")
	dl.traverseFromHead()
	dl.traverseFromTail()
}
