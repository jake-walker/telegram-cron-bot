package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

//var scheduler Scheduler

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v\n", err)
	}

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

		b.Send(m.Sender, "Welcome! Help:\n\n"+
			"*Jobs*\n"+
			"_Jobs are commands that can be run_\n"+
			"/jobs - Get all jobs\n"+
			"/newjob - Create a new job\n"+
			"/deljob - Remove a job\n"+
			"/run - Manually run a job\n"+
			"/set - Set a job environment variable\n"+
			"/get - Get a job environment variable\n\n"+
			"*Tasks*\n"+
			"_Tasks are scheduled jobs_\n"+
			"/tasks - Get all tasks\n"+
			"/newtask - Create a new task\n"+
			"/deltask - Remove a task\n"+
			"/pause - Temporarily stop a task\n"+
			"/resume - Start a paused task\n"+
			"/setoutput - Set the verbosity of a task", tb.ModeMarkdown)
	})

	b.Handle("/jobs", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		jobs, err := AllJobs()
		if err != nil {
			b.Send(m.Sender, "Problem getting jobs")
			return
		}

		if len(jobs) == 0 {
			b.Send(m.Sender, "There are no jobs defined")
			return
		}

		jobsString := ""
		for _, job := range jobs {
			jobsString += fmt.Sprintf("\n- %v\n    Cmd: `%v`\n    Env: `%v`", job.Name, job.Command, strings.Join(job.GetEnv(), ","))
		}
		b.Send(m.Sender, fmt.Sprintf("There are %v jobs:%v", len(jobs), jobsString), tb.ModeMarkdown)
	})

	b.Handle("/newjob", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 2 {
			b.Send(m.Sender, "Usage: /newjob <job name> <command...>")
			return
		}

		job := Job{
			Name:    args[0],
			Command: args[1:],
		}
		err = job.Save()
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error saving job: %v", err))
			return
		}

		b.Send(m.Sender, fmt.Sprintf("New job '%v' created:\n`%v`", job.Name, job.Command), tb.ModeMarkdown)
	})

	b.Handle("/deljob", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 1 {
			b.Send(m.Sender, "Usage: /deljob <job name>")
			return
		}

		job, err := GetJob(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding job: %v", err))
			return
		}

		err = job.Delete()
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error deleting job: %v", err))
			return
		}

		b.Send(m.Sender, fmt.Sprintf("Job '%v' deleted", job.Name))
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

		job, err := GetJob(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding job: %v", err))
			return
		}

		job.run(b, targetChat, OutputFull)
	})

	b.Handle("/tasks", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		tasks, err := AllTasks()
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error getting tasks: %v", err))
			return
		}

		if len(tasks) == 0 {
			b.Send(m.Sender, "There are no tasks scheduled")
			return
		}

		tasksString := ""
		for _, task := range tasks {
			extra := []string{}

			if task.Paused {
				extra = append(extra, "PAUSED")
			}

			tasksString += fmt.Sprintf("\n\n*ID: %v*\n"+
				"_%v_\n"+
				"Job: %v\n"+
				"Cron: `%v`\n"+
				"Output: %v\n"+
				"Next Run: %v", task.Id, strings.Join(extra, ", "), task.JobName, task.Cron, OutputTypeToString(task.OutputType), task.Next.Format(time.UnixDate))
		}
		b.Send(m.Sender, fmt.Sprintf("There are %v tasks scheduled:%v", len(tasks), tasksString), tb.ModeMarkdown)
	})

	b.Handle("/newtask", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 3 {
			b.Send(m.Sender, "Usage: /newtask <job name> once <hhmm> <yyyymmdd: optional>\n"+
				"/newtask <job task> cron <cron expression>")
			return
		}

		job, err := GetJob(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding job: %v", err))
			return
		}

		task := Task{
			JobName:    job.Name,
			OutputType: OutputFull,
		}

		if args[1] == "once" {
			rawTime := args[2]
			rawDate := ""

			if len(args) < 4 || args[3] == "" {
				rawDate = time.Now().Format("20060102")
			}

			loc, err := time.LoadLocation(config.Timezone)
			if err != nil {
				b.Send(m.Sender, fmt.Sprintf("Error loading timezone: %v", err))
				return
			}

			jobDate, err := time.ParseInLocation("1504 20060102", rawTime+" "+rawDate, loc)
			if err != nil {
				b.Send(m.Sender, fmt.Sprintf("Error parsing date: %v", err))
				return
			}

			task.Next = jobDate
		} else if args[1] == "cron" {
			task.Cron = strings.Join(args[2:], " ")
			scheduled, err := task.Reschedule(config.Timezone)
			if err != nil || !scheduled {
				b.Send(m.Sender, fmt.Sprintf("Could not create task: %v", err))
				return
			}
		} else {
			b.Send(m.Sender, "Whoops! The second argument must be once or cron")
			return
		}

		task.Save()
		b.Send(m.Sender, fmt.Sprintf("Task %v is scheduled for %v", task.Id, task.Next.Format(time.UnixDate)))
	})

	b.Handle("/deltask", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 1 {
			b.Send(m.Sender, "Usage: /deltask <id>")
			return
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error parsing task id: %v", err))
			return
		}

		task, err := GetTask(id)
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding task: %v", err))
			return
		}

		err = task.Delete()
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error deleting task: %v", err))
			return
		}

		b.Send(m.Sender, fmt.Sprintf("Task %v deleted", task.Id))
	})

	b.Handle("/pause", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 1 {
			b.Send(m.Sender, "Usage: /pause <id>")
			return
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error parsing task id: %v", err))
			return
		}

		task, err := GetTask(id)
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding task: %v", err))
			return
		}

		err = task.Pause(true)
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error deleting task: %v", err))
			return
		}

		b.Send(m.Sender, fmt.Sprintf("Task %v paused", task.Id))
	})

	b.Handle("/resume", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 1 {
			b.Send(m.Sender, "Usage: /resume <id>")
			return
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error parsing task id: %v", err))
			return
		}

		task, err := GetTask(id)
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding task: %v", err))
			return
		}

		err = task.Pause(false)
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error deleting task: %v", err))
			return
		}

		rescheduled, err := task.Reschedule(config.Timezone)
		if err != nil || !rescheduled {
			log.Printf("error rescheduling task %v: %v\n", task.Id, err)
		}

		b.Send(m.Sender, fmt.Sprintf("Task %v resumed", task.Id))
	})

	b.Handle("/set", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 3 {
			b.Send(m.Sender, "Usage: /set <job name> <key> <value>")
			return
		}

		job, err := GetJob(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding job: %v", err))
			return
		}

		key := strings.TrimSpace(args[1])
		val := strings.TrimSpace(args[2])
		if key == "" || val == "" {
			b.Send(m.Sender, "The key or value is blank")
			return
		}

		if job.Env == nil {
			job.Env = make(map[string]string)
		}

		job.Env[args[1]] = args[2]
		err = job.Save()
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error saving job: %v", err))
			return
		}

		b.Send(m.Sender, fmt.Sprintf("'%v' has been set to '%v' on job '%v'", args[1], args[2], job.Name))
	})

	b.Handle("/get", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 1 {
			b.Send(m.Sender, "Usage: /get <job_name> <key?>")
			return
		}

		job, err := GetJob(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding job: %v", err))
			return
		}

		if job.Env == nil {
			job.Env = make(map[string]string)
		}

		if len(args) < 2 {
			output := fmt.Sprintf("Job '%v' has the following environment variables:\n\n", job.Name)
			for k, v := range job.Env {
				output = output + fmt.Sprintf("- '%v' = '%v'\n", k, v)
			}
			b.Send(m.Sender, output)
			return
		}

		val, set := job.Env[args[1]]
		if !set {
			b.Send(m.Sender, fmt.Sprintf("Job '%v' does not have '%v' set", job.Name, args[1]))
			return
		}

		b.Send(m.Sender, fmt.Sprintf("Job '%v' has '%v' set as '%v'", job.Name, args[1], val))
	})

	b.Handle("/setoutput", func(m *tb.Message) {
		if m.Chat.ID != targetChat.ID {
			b.Send(m.Sender, "Whoops! You are not authorized to use this bot")
			return
		}

		args := strings.Split(m.Payload, " ")
		if len(args) < 2 {
			b.Send(m.Sender, "Usage: /setoutput <id> <type>")
			return
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error parsing task id: %v", err))
			return
		}

		task, err := GetTask(id)
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error finding task: %v", err))
			return
		}

		switch strings.ToLower(strings.TrimSpace(args[1])) {
		case "none":
			task.OutputType = OutputNone
		case "result":
			task.OutputType = OutputResult
		case "last":
			task.OutputType = OutputLastLine
		case "all":
			task.OutputType = OutputFull
		default:
			b.Send(m.Sender, "Unknown output type. It can be 'none', 'result', 'last' or 'all'.")
			return
		}

		err = task.Save()
		if err != nil {
			b.Send(m.Sender, fmt.Sprintf("Error saving task: %v", err))
			return
		}

		b.Send(m.Sender, fmt.Sprintf("Task %v output set to %v", task.Id, OutputTypeToString(task.OutputType)))
	})

	go func(cfg *Config, bot *tb.Bot, targetChat *tb.Chat) {
		for range time.Tick(time.Second * 30) {
			CheckTasks(bot, targetChat, config)
		}
	}(&config, b, targetChat)

	log.Println("starting...")
	b.Send(targetChat, "Bot started!")
	b.Start()
}

func (j *Job) run(b *tb.Bot, targetChat *tb.Chat, outputType JobOutputType) {
	cmd := exec.Command(j.Command[0], j.Command[1:]...)
	cmd.Env = j.GetEnv()
	out, err := cmd.CombinedOutput()

	if err != nil && outputType == OutputNone {
		outputType = OutputLastLine
	}

	msg := ""

	if err == nil {
		msg = msg + "✅ "
	} else {
		msg = msg + "❌ "
	}

	if outputType == OutputFull || outputType == OutputResult {
		if err != nil {
			msg = msg + fmt.Sprintf("Job '%v' failed: %v", j.Name, err)
		} else {
			msg = msg + fmt.Sprintf("Job '%v' completed", j.Name)
		}
	}

	if outputType == OutputFull {
		msg = msg + fmt.Sprintf("\n\n%v", string(out))
	}

	if outputType == OutputLastLine {
		lines := strings.Split(string(out), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				msg = msg + lines[i]
				break
			}
		}
	}

	if outputType != OutputNone {
		b.Send(targetChat, msg)
	}
}
