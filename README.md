# Redsync Testing

So this is a simple test of the Redis Redsync mutex -- except, this has exponential backoff and context retry logic (maybe not so simple lol). To understand the exponential backoff, `go run backoff/main.go` and make sure to play with the settings.

There is some crazy read the hostname and parse this into a rand.Seed seeder (lol) in order to assure we do not get split brain in the case all the go binaries try to lock at the same exact time. The hostname is meant to be used with containers -- no 2 containers will have the same hostname value (or should not).

```bash
$ docker-compose -f redsync/docker-compose.yml up --build

### To destroy the containers...
$ docker-compose -f redsync/docker-compose.yml down
```