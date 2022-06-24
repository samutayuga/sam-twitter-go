package main

import (
	assert2 "github.com/stretchr/testify/assert"
	"testing"
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
