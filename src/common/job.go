package common

import (
	"bytes"
	"encoding/json"
	"time"
)

const (
	PRIORITY_LOW  = 0
	PRIORITY_HIGH = 1
	JobPrefix     = "H:"
)

type Job struct {
	Handle       string //server job handle
	Id           string
	Data         []byte
	Running      bool
	Percent      int
	Denominator  int
	CreateAt     time.Time
	ProcessAt    time.Time
	TimeoutSec   int
	CreateBy     int64 //client sessionId
	ProcessBy    int64 //worker sessionId
	FuncName     string
	IsBackGround bool
	Priority     int
}

func (job *Job) String() string {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	m := make(map[string]interface{})
	m["Handle"] = job.Handle
	m["Id"] = job.Id
	m["Data"] = string(job.Data)
	m["Running"] = job.Running
	m["Percent"] = job.Percent
	m["Denominator"] = job.Denominator
	m["CreateAt"] = job.CreateAt
	m["ProcessAt"] = job.ProcessAt
	m["Running"] = job.Running
	m["TimeoutSec"] = job.TimeoutSec
	m["CreateBy"] = job.CreateBy
	m["ProcessBy"] = job.ProcessBy
	m["FuncName"] = job.FuncName
	m["IsBackGround"] = job.IsBackGround
	m["Priority"] = job.Priority

	if err := enc.Encode(m); err != nil {
		return ""
	}

	return string(b.Bytes())
}
