package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	nanoid "github.com/matoous/go-nanoid/v2"
	tb "gopkg.in/tucnak/telebot.v2"
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

// Schedule a task without locking the scheduler
func (s *Scheduler) scheduleRaw(task Task) string {
	id, err := nanoid.Generate("abcdefghijklmnopqrstuvwxyz", 5)
	s.tasks[id] = task
	if err != nil {
		log.Fatalf("error generating id: %v\n", err)
	}
	log.Printf("task %v: scheduled job %v for %v\n", id, task.JobName, task.Date)
	return id
}

// Regular way of scheduling a task
func (s *Scheduler) Schedule(task Task) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.scheduleRaw(task)
	err := s.save()
	return id, err
}

// Unschedule a task without locking the scheduler
func (s *Scheduler) unscheduleRaw(id string) {
	delete(s.tasks, id)
	log.Printf("task %v: unscheduled", id)
}

// Regular way of unscheduling a task
func (s *Scheduler) Unschedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unscheduleRaw(id)
	err := s.save()
	return err
}

func (s *Scheduler) CheckTasks(config *Config, bot *tb.Bot, chat *tb.Chat) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//log.Println("checking schedule...")
	now := time.Now()

	// are changes made to the schedule?
	changes := false

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
			s.unscheduleRaw(id)
			changes = true
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
			changes = true
		} else {
			log.Print("    removing\n")
			s.unscheduleRaw(id)
			changes = true
		}
	}

	if changes {
		err := s.save()
		if err != nil {
			log.Printf("error saving schedule: %v\n", err)
			return
		}
	}
}

func (s *Scheduler) save() error {
	log.Println("saving schedule...")

	data, err := json.Marshal(s.tasks)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(ConfigDirectory("schedule.json"), data, 0600)
	return err
}

func (s *Scheduler) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("loading schedule...")

	data, err := ioutil.ReadFile(ConfigDirectory("schedule.json"))
	if os.IsNotExist(err) {
		log.Println("schedule.json does not exist")
		return nil
	} else if err != nil {
		return err
	}

	err = json.Unmarshal(data, &s.tasks)
	return err
}
