# GitLab Runner Droplet Monitor [![pipeline status](https://git.ethitter.com/debian/gitlab-runner-do-monitor/badges/master/pipeline.svg)](https://git.ethitter.com/debian/gitlab-runner-do-monitor/commits/master)

Monitor Digital Ocean for stale droplets created by GitLab Runner

## Configuration

```json
{
  "log-dest": "os.Stdout",
  "api-key": "",
  "threshold": 5400,
  "delete-stale": true
}

```

* `log-dest`: set to a path to write to a log file, otherwise `os.Stdout`
* `api-key`: Digital Ocean Personal Access Token
* `threshold`: time, in seconds, after which to consider a runner stale
* `delete-stale`: whether to delete stale runners, in addition to reporting them

##### A note about `threshold`

This value needs to be greater than the job timeout specified in your GitLab Runner configuration, otherwise a runner may erroneously be considered stale.

## Installation

1. Download the appropriate binary from [tagged releases](https://git.ethitter.com/debian/gitlab-runner-do-monitor/tags), or build the binary yourself.
1. Copy `config-sample.json` to an appropriate location and update the default values as needed.
1. Create a cron task to periodically run the monitor.

## Usage

```bash
./glrdomon -config config.json
```

* `-config`: specify path to config file, otherwise assumes `./config.json` relative to the binary
