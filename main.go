package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

var scheduler Scheduler

func getJob(config *Config, name string) (found bool, index int) {
	found = false

	for i, job := range config.Jobs {
		if job.Name == name {
			index = i
			found = true
			return
		}
	}

	return
}

func main() {
	scheduler = *NewScheduler()
	err := scheduler.Load()
	if err != nil {
		log.Fatalf("failed to load scheduler.json: %v\n", err)
	}

	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v\n", err)
	}
	log.Printf("found %v jobs", len(config.Jobs))

	b, err := tb.NewBot(tb.Settings{
		Token:  config.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalln(err)
	}

	targetChat, err := b.ChatByID(config.ChatId)
	if err != nil {
		log.Fatalln(err)
	}

	b.Handle("/start", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		b.Send(m.Sender, "Welcome!\n"+
			"Use /jobs to see available jobs")
	})

	b.Handle("/jobs", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		if len(config.Jobs) == 0 {
			b.Send(m.Sender, "There are no jobs defined")
			return
		}

		jobsString := ""
		for _, job := range config.Jobs {
			jobsString += fmt.Sprintf("\n- %v", job.Name)
		}
		b.Send(m.Sender, fmt.Sprintf("There are %v jobs:%v", len(config.Jobs), jobsString))
	})

	b.Handle("/run", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")

		if len(args) < 1 {
			b.Send(m.Sender, "Usage: /run <job name>")
			return
		}

		found, jobIndex := getJob(&config, args[0])

		if !found {
			b.Send(m.Sender, fmt.Sprintf("Job '%v' was not found", args[0]))
			return
		}

		config.Jobs[jobIndex].run(b, targetChat)
	})

	b.Handle("/tasks", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		if len(scheduler.tasks) == 0 {
			b.Send(m.Sender, "There are no tasks scheduled")
			return
		}

		tasksString := ""
		for id, task := range scheduler.tasks {
			if task.Next > 0 {
				tasksString += fmt.Sprintf("\n\n**%v**\nID: %v\nJob: %v\nRepeats every %v", task.Date.Format(time.UnixDate), id, task.JobName, task.Next.String())
			} else {
				tasksString += fmt.Sprintf("\n\n**%v**\nID: %v\nJob: %v\nDoes not repeat", task.Date.Format(time.UnixDate), id, task.JobName)
			}
		}
		b.Send(m.Sender, fmt.Sprintf("There are %v tasks scheduled:%v", len(scheduler.tasks), tasksString), tb.ModeMarkdownV2)
	})

	b.Handle("/schedule", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")

		if len(args) < 2 {
			b.Send(m.Sender, "Usage: /schedule <job name> <time: hhmm> <date? yyyymmdd (default: today)> <interval? (default: none)>")
			return
		}

		found, _ := getJob(&config, args[0])

		if !found {
			b.Send(m.Sender, fmt.Sprintf("Job '%v' was not found", args[0]))
			return
		}

		rawTime := args[1]
		rawDate := ""

		if len(args) > 2 {
			rawDate = args[2]
		}

		if len(rawTime) != 4 {
			b.Send(m.Sender, "Time is in an invalid format, use 24h in the format hhmm")
			return
		}

		if rawDate == "" {
			rawDate = time.Now().Format("20060102")
		} else if len(rawDate) != 8 {
			b.Send(m.Sender, "Date is an invalid format, use the format yyyymmdd")
			return
		}

		loc, err := time.LoadLocation(config.Timezone)
		if err != nil {
			b.Send(m.Sender, "Error loading timezone")
			log.Printf("error loading timezone: %v", err)
		}

		jobDate, err := time.ParseInLocation("1504 20060102", rawTime+" "+rawDate, loc)

		if err != nil {
			b.Send(m.Sender, "Error parsing date")
			log.Printf("error parsing date: %v", err)
			return
		}

		var interval time.Duration

		if len(args) > 3 && args[3] != "" {
			interval, err = time.ParseDuration(args[3])
			if err != nil {
				b.Send(m.Sender, "Error parsing interval")
				log.Printf("error parsing interval: %v", err)
				return
			}
		}

		scheduler.Schedule(Task{
			Date:    jobDate,
			JobName: args[0],
			Next:    interval,
		})

		repeatText := ""
		if interval > 0 {
			repeatText = fmt.Sprintf(" and repeats every %v", interval.String())
		}
		b.Send(m.Sender, fmt.Sprintf("Job '%v' scheduled for %v%v", args[0], jobDate.Format(time.UnixDate), repeatText))
	})

	b.Handle("/unschedule", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")

		if len(args) < 1 {
			b.Send(m.Sender, "Usage: /unschedule <task id>")
			return
		}

		err := scheduler.Unschedule(args[0])
		if err != nil {
			b.Send(m.Sender, "Error unscheduling")
			log.Printf("error unscheduling: %v\n", err)
			return
		}

		b.Send(m.Sender, fmt.Sprintf("Unscheduled task '%v'", args[0]))
	})

	go func(cfg *Config, bot *tb.Bot, targetChat *tb.Chat, sch *Scheduler) {
		for range time.Tick(time.Second * 30) {
			sch.CheckTasks(cfg, bot, targetChat)
		}
	}(&config, b, targetChat, &scheduler)

	log.Println("starting...")
	b.Send(targetChat, fmt.Sprintf("Bot started with %v jobs and %v scheduled tasks", len(config.Jobs), len(scheduler.tasks)))
	b.Start()
}

func (j *Job) run(b *tb.Bot, targetChat *tb.Chat) {
	b.Send(targetChat, fmt.Sprintf("Running job '%v'...", j.Name))
	out, err := exec.Command(j.Command[0], j.Command[1:]...).CombinedOutput()
	if err != nil {
		b.Send(targetChat, fmt.Sprintf("Job '%v' failed: `%v`", j.Name, err), tb.ModeMarkdownV2)
	} else {
		b.Send(targetChat, fmt.Sprintf("Job '%v' successful", j.Name))
	}
	b.Send(targetChat, "```\n"+string(out)+"\n```", tb.ModeMarkdownV2)
}
