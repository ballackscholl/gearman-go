package memory

import (
	. "common"
	"bytes"
	"fmt"
	"container/list"
)

type MemJobQueue struct {
	name  string
	queue *list.List
}

func (m *MemJobQueue) Initial(name string) {

	m.name = name
	m.queue = list.New()

}

func (m *MemJobQueue) PushJob(job *Job) {

	if job != nil {
		m.queue.PushBack(job)
	}
}

func (m *MemJobQueue) PopJob() *Job {

	element := m.queue.Back()
	if element != nil {
		job := element.Value.(*Job)
		m.queue.Remove(element)
		return job
	}
	return nil
}

func (m *MemJobQueue) RemoveJob(handle string) *Job {

	var job *Job = nil

	for e := m.queue.Front(); e != nil; e = e.Next() {
		if e.Value.(*Job).Handle == handle {
			job = e.Value.(*Job)
			m.queue.Remove(e)
			break
		}
	}

	return job
}

func (m *MemJobQueue) Show() string{

	var buffer bytes.Buffer

	for e := m.queue.Front(); e != nil; e = e.Next() {
		buffer.WriteString(fmt.Sprintf("%v\n", e))
	}

	return buffer.String()
}

func (m *MemJobQueue) Length() int {
	return m.queue.Len()
}
