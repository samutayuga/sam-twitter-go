package main

import (
	assert2 "github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

var testDataForPush = []struct {
	expected      string
	elementPushed string
}{
	{STARTED, STARTED},
	{STOPPED, STOPPED},
}

func TestPush(t *testing.T) {
	stateStacks := StateStack{}
	stateStacks.New()
	assert := assert2.New(t)
	for _, dataSet := range testDataForPush {
		stateStacks.Push(dataSet.elementPushed)
		assert.Equal(dataSet.expected, stateStacks.Pop())
		assert.True(0 == len(stateStacks.items))
	}

}

func TestPop(t *testing.T) {
	stateStacks := StateStack{}
	stateStacks.New()
	stateStacks.Push(STARTED)
	stateStacks.Push(PAUSED)
	stateStacks.Push(STOPPED)
	assert := assert2.New(t)
	assert.Equal(STOPPED, stateStacks.Pop())
	assert.True(2 == len(stateStacks.items))
}

var date = []struct {
	dateString string
}{
	{"2019-10-01T00:00:00+00:00"},
	{"2019-10-01T00:00:04+00:00"},
	{"2019-10-01T00:00:08+00:00"},
	{"2019-10-01T00:00:24+00:00"},
}

func TestEnqueue(t *testing.T) {
	//set 5 months ago
	dateQueue := DateQueue{}
	dateQueue.New()
	for _, dateStr := range date {
		if past, err := time.Parse(time.RFC3339, dateStr.dateString); err == nil {
			diff := time.Since(past)
			dateQueue.Enqueu(diff)
			x := dateQueue.Dequeue()

			log.Printf("diff=%v", diff)
			assert := assert2.New(t)
			assert.True(x != nil)
		}
	}

}
