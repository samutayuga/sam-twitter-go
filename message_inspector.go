package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

//Node is structure to point to the next element
type Node struct {
	Data time.Duration
	//Next is pointing to the next element
	Next *Node
	//Previous is pointing to the previous element
	Previous *Node
}

//DurationQueue a queue to manage the duration
type DurationQueue struct {
	lock sync.RWMutex
	dl   DoublyLinkedList
}

//New instantiates the queue based on the doubly linked list data structure
func (q *DurationQueue) New() {
	q.lock.Lock()
	q.dl = DoublyLinkedList{}
	q.lock.Unlock()
}

//Enqueue enqueue the new duration
func (q *DurationQueue) Enqueue(duration time.Duration) {
	q.lock.Lock()
	q.dl.Insert(duration)
	q.lock.Unlock()
}

//Dequeue get and delete the
func (q *DurationQueue) Dequeue() *time.Duration {
	q.lock.Lock()
	del := q.dl.Delete()
	q.lock.Unlock()
	if del != nil {
		return &del.Data
	}
	return nil

}

//DoublyLinkedList a structure that gives better running time complexity when inserting the new data and delete
//It is O(1) constant running complexity when insert
//compare to O(N) linear when insert into array
//
type DoublyLinkedList struct {
	//Head in a linked list data type point to the head of the node
	Head *Node
	//Tail in a linked list data type point to the last element of the node
	Tail *Node
}

func (d *DoublyLinkedList) Insert(duration time.Duration) {
	aNode := Node{
		Data:     duration,
		Next:     nil,
		Previous: nil,
	}
	//1. check if it is the first node
	if d.Head == nil {
		d.Head = &aNode
		d.Tail = &aNode
		//d.Tail.Previous = d.Head
		//d.Head.Next = d.Tail

	} else {
		// this is when the structure is not an empty
		//2. previous tail is new node's previous node
		aNode.Previous = d.Tail
		d.Tail.Next = &aNode
		//3. current tail is new node
		d.Tail = &aNode
	}
}
func traverse(aNode *Node) {
	fmt.Printf("%s\n", aNode.Data.String())
	if aNode.Next != nil {
		traverse(aNode.Next)
	}
}
func traverseBack(aNode *Node) {
	fmt.Printf("%s\n", aNode.Data.String())
	if aNode.Previous != nil {
		traverseBack(aNode.Previous)
	}
}
func (d *DoublyLinkedList) traverseFromHead() {
	traverse(d.Head)
}
func (d *DoublyLinkedList) traverseFromTail() {
	traverseBack(d.Tail)
}

//Delete the item from end of the structure
func (d *DoublyLinkedList) Delete() *Node {

	if d.Tail != nil {
		newTail := d.Tail.Previous
		deleted := d.Tail
		if newTail != nil {
			d.Tail = newTail
			prevNode := newTail
			//tail next node is nil
			prevNode.Next = nil
		} else {
			//this is the first element
			d.Head = nil
			d.Tail = nil
		}
		return deleted
	} else {
		return nil
	}

}

//StateStack is a stack of items
type StateStack struct {
	items []string
	lock  sync.RWMutex
}

//New is a function to create a new stack of item
func (s *StateStack) New() *StateStack {
	//1. simply create an empty slices of Item
	//2. stack is initialised with the empty slice
	s.items = []string{}
	return s
}

//Push is a function to add a new item into the stack
func (s *StateStack) Push(t string) {
	//3. lock before append to prevent different goroutine update it
	s.lock.Lock()
	s.items = append(s.items, t)
	//4. unlock before append to prevent different goroutine update it
	s.lock.Unlock()
}

//Pop is a function to get and remove a new item from the stack
//remember the LIFO behavior of the stack
func (s *StateStack) Pop() string {
	//5. lock before get to prevent different goroutine update it
	s.lock.Lock()
	//6. get the item with the greatest index
	item := s.items[len(s.items)-1]
	//7. make a new slice where the last index is excluded
	s.items = s.items[0 : len(s.items)-1]
	//8. unlock before append to prevent different goroutine update it
	s.lock.Unlock()
	//9. return
	return item
}

//Peek is to check the current state without removing it
//from the stack
//this is read operation so no locking is needed
func (s *StateStack) Peek() string {
	return s.items[len(s.items)-1]
}

//DateQueue is a queue of date
type DateQueue struct {
	dates []time.Duration
	lock  sync.RWMutex
}

func (q *DateQueue) New() *DateQueue {
	q.dates = []time.Duration{}
	return q
}
func (q *DateQueue) Enqueu(aDate time.Duration) {
	q.lock.Lock()
	q.dates = append(q.dates, aDate)
	q.lock.Unlock()
}
func (q *DateQueue) Dequeue() *time.Duration {
	q.lock.Lock()
	if len(q.dates) > 0 {
		aDate := q.dates[0]
		q.dates = q.dates[1:len(q.dates)]
		q.lock.Unlock()
		return &aDate
	}
	q.lock.Unlock()
	return nil
}

func doConsume(tm *time.Duration, durationQueue DurationQueue, completed chan bool) {
	//var nextTm *time.Duration
	var diff float64
	if nextTm := durationQueue.Dequeue(); nextTm != nil {

		//compare
		diff = nextTm.Seconds() - tm.Seconds()
		log.Printf("diff %f\n", diff)
		doConsume(nextTm, durationQueue, completed)
	} else {
		completed <- true
	}
}
func ConsumeQueue(durationQueue DurationQueue, completed chan bool) {
	//first content
	if firstTime := durationQueue.Dequeue(); firstTime != nil {
		doConsume(firstTime, durationQueue, completed)
	} else {
		completed <- true
	}

}
