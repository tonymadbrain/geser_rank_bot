# Geser Rank Bot

Geser is a Telegram bot for user ratings in Chat Groups. He increments rank for a person when a reply to his message starts with a `+` sign. To start, add @geser_rank_bot to your Telegram Chat Group.

## Features

- [x] react on messages starting with +
- [x] increment rank
- [x] dump rank to disk periodically
- [x] load dump from disk on start
- [x] /top_rank command for top 10 users in group

## TODO

- [ ] unit testing
- [ ] log when bot added to new group
- [ ] conteinerize?
- [ ] releases?
- [ ] add metrics sending (influx?), user/group counts
- [ ] support rank decrement optionally (one minus not equal to one plus)
- [ ] react on words array to force using +
- [ ] detailed stats for user, private with info about all groups

## How to run

Clone project

```Bash
$ git clone git@github.com:tonymadbrain/geser_rank_bot.git
$ cd geser_rank_bot
```

Install dependencies
```Bash
$ go mod vendor
```

Build for you arch
```Bash
$ env GOOS=linux GOARCH=amd64 go build .
```

Run
```Bash
$ TG_BOT_TOKEN=1234567890:xxxYYYzzz ./geser_rank_bot
```

To run as a daemon use gesera_rank_bot.service as example for Systemd.

