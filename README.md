# Telegram Cron Bot

A work-in-progress bot for running scheduled tasks through Telegram.

## Config

This bot requires a config file in the working directory named `config.yml`:

```yaml
token: "<<TELEGRAM_BOT_TOKEN>>"
chat_id: "<<TELEGRAM_USER_ID>>"
timezone: "Europe/London"
jobs:
  - name: "test"
    command: ["ls", "-al"]
  - name: "pwd"
    command: ["pwd"]
```

## Commands

- `/jobs` - View jobs that are defined in the config file.
- `/run <jobname>` - Run a job immediately once.
- `/tasks` - View scheduled jobs.
- `/schedule <jobname> <time>` - Run a job at a certain time (`hhmm`) **today**.
  _e.g. `/schedule pwd 1530` - run 'pwd' at 3:30pm today._
- `/schedule <jobname> <time> <date>` - Run a job at a certain time (`hhmm`) on a certain date (`yyyymmdd`).
  _e.g. `/schedule test 0000 20220101` - run 'test' at midnight on 01/01/2022._
- `/schedule <jobname> <time> <date> <interval>` - Same as above, but once completed, reschedule for interval in the future.
  _e.g. `/schedule test 1121 20210710 1h` - run 'test' at 11:21am on 10/07/2021 and then every hour afterwards._ 