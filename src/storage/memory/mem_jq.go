package memory

import (
	. "common"
	"container/list"
	"sync"
)

type MemJobQueue struct {
	name  string
	queue *list.List
	qLock sync.Mutex
}

func (m *MemJobQueue) Initial(name string) {

	m.name = name
	m.queue = list.New()

}

func (m *MemJobQueue) PushJob(job *Job) {
	m.qLock.Lock()
	defer m.qLock.Unlock()

	if job != nil {
		m.queue.PushBack(job)
	}
}

func (m *MemJobQueue) PopJob() *Job {

	m.qLock.Lock()
	defer m.qLock.Unlock()

	element := m.queue.Back()
	if element != nil {
		job := element.Value.(*Job)
		m.queue.Remove(element)
		return job
	}
	return nil
}

func (m *MemJobQueue) RemoveJob(handle string) *Job {

	m.qLock.Lock()
	defer m.qLock.Unlock()

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

func (m *MemJobQueue) Length() int {

	return m.queue.Len()

}
