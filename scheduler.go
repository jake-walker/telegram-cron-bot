package main

import (
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func CheckTasks(bot *tb.Bot, chat *tb.Chat, config Config) {
	log.Println("checking schedule...")
	now := time.Now()

	tasks, err := AllTasks()
	if err != nil {
		log.Printf("error fetching tasks: %v\n", err)
	}

	for _, task := range tasks {
		// this task is in the future or paused, skip it
		if task.Next.After(now) || task.Paused {
			continue
		}

		log.Printf(" -> task: %v, job: %v\n", task.Id, task.JobName)

		log.Print("    finding job\n")

		job, err := GetJob(task.JobName)
		if err != nil {
			log.Printf("error finding job %v: %v\n", task.JobName, err)
			continue
		}

		log.Print("    running job\n")
		job.run(bot, chat, task.OutputType)

		if task.Cron == "" {
			// task done, unschedule
			log.Printf("task %v unscheduled\n", task.Id)
			task.Delete()
			continue
		}

		rescheduled, err := task.Reschedule(config.Timezone)
		if err != nil || !rescheduled {
			log.Printf("error rescheduling task %v: %v\n", task.Id, err)
			continue
		}
	}
}
