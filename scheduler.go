package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/rs/xid"
)

type Task struct {
	Date    time.Time     `json:"date"`
	Next    time.Duration `json:"next"`
	JobName string        `json:"job"`
}

type Scheduler struct {
	mu    sync.Mutex
	tasks map[string]Task
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make(map[string]Task),
	}
}

func (s *Scheduler) GetTasks() map[string]Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.tasks
}

func (s *Scheduler) ScheduleNoLock(task Task) string {
	id := xid.New().String()
	s.tasks[id] = task
	log.Printf("task %v: scheduled job %v for %v\n", id, task.JobName, task.Date)
	return id
}

func (s *Scheduler) Schedule(task Task) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ScheduleNoLock(task)
}

func (s *Scheduler) UnscheduleNoLock(id string) {
	delete(s.tasks, id)
	log.Printf("task %v: unscheduled", id)
}

func (s *Scheduler) Unschedule(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.UnscheduleNoLock(id)
}

func (s *Scheduler) CheckTasks(config *Config, bot *tb.Bot, chat *tb.Chat) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//log.Println("checking schedule...")
	now := time.Now()

	for id, task := range s.tasks {
		// this task is in the future, skip it
		if task.Date.After(now) {
			continue
		}

		log.Printf(" -> task: %v, job: %v\n", id, task.JobName)

		log.Print("    finding job\n")

		found, jobIndex := getJob(config, task.JobName)

		if !found {
			log.Printf("    job not found, unscheduling")
			s.UnscheduleNoLock(id)
			continue
		}

		log.Print("    running job\n")
		config.Jobs[jobIndex].run(bot, chat)

		if task.Next > 0 {
			nextDate := task.Date.Add(task.Next)
			log.Printf("    rescheduling for %v\n", nextDate)
			bot.Send(chat, fmt.Sprintf("Job '%v' has been rescheduled for %v", task.JobName, nextDate.Format(time.UnixDate)))

			newTask := s.tasks[id]
			newTask.Date = nextDate
			s.tasks[id] = newTask
		} else {
			log.Print("    removing\n")
			s.UnscheduleNoLock(id)
		}
	}
}

func (s *Scheduler) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(s.tasks)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("schedule.json", data, 0600)
	return err
}

func (s *Scheduler) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := ioutil.ReadFile("schedule.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &s.tasks)
	return err
}
