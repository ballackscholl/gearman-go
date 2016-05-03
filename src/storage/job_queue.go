package storage

import (
	. "common"
)

type JobQueue interface {
	Initial(name string)
	PushJob(job *Job)
	PopJob() *Job
	RemoveJob(handle string) *Job
	Length() int
	Show() string
}
