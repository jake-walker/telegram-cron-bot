# Telegram Cron Bot

A work-in-progress bot for running scheduled tasks through Telegram.

## Deployment

### Docker

The recommended way of running the bot is in a Docker container. Simply run `docker run -it --restart=unless-stopped -v ${HOME}/bot.yml:/config/config.yml ghcr.io/jake-walker/telegram-cron-bot:latest`.

If you want to add extra stuff into the container, for example, a Python script, you will need to make your own container with a `Dockerfile` like this:

```dockerfile
FROM ghcr.io/jake-walker/telegram-cron-bot:latest

RUN apk update && apk add --no-cache python3

RUN mkdir -p /jobs/example
COPY ./myscript.py /jobs/example/
```

### Manually

You can build the bot manually by running `go build -o bot .`.

## Config

This bot requires a config file in the working directory named `config.yml`:

```yaml
token: "<<TELEGRAM_BOT_TOKEN>>"
chat_id: "<<TELEGRAM_USER_ID>>"
timezone: "Europe/London"
```

## Usage & Commands

### Jobs

Jobs define a single command that will get run.

- `/jobs` - View all the currently set up jobs.
- `/newjob <name> <command...>` - Create a new job.
- `/deljob <name>` - Delete a job.
- `/run <name>` - Manually run a job once.

Examples:

- `/newjob example curl https://example.com` - Create a new job which makes a web request to example.com.
- `/deljob example` - Delete the example job.
- `/run example` - Run the example job.

### Tasks

Tasks schedule a job (or command) which can run once in the future or repeatedly with a cron pattern.

- `/tasks` - View all the currently set up tasks.
- `/newtask <job> once <hhmm>` - Run a job once at a set time today.
- `/newtask <job> once <hhmm> <yyyymmdd>` - Run a job once at a set time and time.
- `/newtask <job> cron <expr>` - Run a job in a repeating pattern according to a cron expression.

Examples:

- `/newtask example once 1545` - Run the example job at 3:45pm today.
- `/newtask example once 1900 20220101` - Run the example job at 7pm on the 1st January 2022.
- `/newtask example cron 15 17 * * 1-5` - Run the example job at 5:15pm Monday to Friday.
