package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/asdine/storm"
)

type Job struct {
	Name    string `storm:"id"`
	Command []string
	Env     map[string]string
}

type Task struct {
	Id      int `storm:"id,increment"`
	Cron    string
	Next    time.Time
	JobName string
	Verbose bool
	Paused  bool
}

// Returns a bool which is true if rescheduled
func (t *Task) Reschedule(timezone string) (bool, error) {
	if t.Cron == "" {
		return false, nil
	}

	now := time.Now().UTC()

	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return false, err
		}
		now = now.In(loc)
	}

	nextTime := cronexpr.MustParse(t.Cron).Next(now)

	if nextTime.IsZero() {
		return false, errors.New("zero date")
	}

	db, err := getDb()
	defer db.Close()
	if err != nil {
		return false, err
	}
	if t.Id > 0 {
		err = db.UpdateField(t, "Next", nextTime)
		return true, err
	} else {
		t.Next = nextTime
		log.Printf("not saving task %v\n", t.Id)
		return true, nil
	}
}

func getDb() (*storm.DB, error) {
	return storm.Open(ConfigDirectory("cron.db"))
}

func (j *Job) Save() error {
	db, err := getDb()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Save(j)
}

func (j *Job) Delete() error {
	db, err := getDb()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.DeleteStruct(j)
}

func (j *Job) GetEnv() []string {
	output := []string{}

	if j.Env == nil {
		return output
	}

	for k, v := range j.Env {
		output = append(output, fmt.Sprintf("%v=%v", strings.ToUpper(k), v))
	}

	return output
}

func (t *Task) Save() error {
	db, err := getDb()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Save(t)
}

func (t *Task) Delete() error {
	db, err := getDb()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.DeleteStruct(t)
}

func (t *Task) Pause(paused bool) error {
	db, err := getDb()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.UpdateField(t, "Paused", paused)
}

func AllJobs() ([]Job, error) {
	var jobs []Job

	db, err := getDb()
	if err != nil {
		return jobs, err
	}
	defer db.Close()

	err = db.All(&jobs)
	return jobs, err
}

func GetJob(name string) (Job, error) {
	var job Job

	db, err := getDb()
	if err != nil {
		return job, err
	}
	defer db.Close()

	err = db.One("Name", name, &job)
	return job, err
}

func AllTasks() ([]Task, error) {
	var tasks []Task

	db, err := getDb()
	if err != nil {
		return tasks, err
	}
	defer db.Close()

	err = db.All(&tasks)
	return tasks, err
}

func GetTask(id int) (Task, error) {
	var task Task

	db, err := getDb()
	if err != nil {
		return task, err
	}
	defer db.Close()

	err = db.One("Id", id, &task)
	return task, err
}
