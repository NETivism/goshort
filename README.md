# goshort - shorten url service in go

## Build from source

This will using docker compose to build base on Dockerfile. And run the service.

The database file will be locate at directory `./docker/_yourdbname_.sqlite`.
The docker image name will be tag by netivism/goshort:local
```
cd docker
docker compose -f docker-compose-src.yml build
```

## Run service

### Run from built binary
Following above step, you will get image netivism/goshort:local. Use `up` for running that image
```
docker compose -f docker-compose-src.yml up
```

### Run from remote container

The default docker compose will use image netivism/goshort:sqlite
```
docker compose up
```
Check `./docker/docker-compose.yml` for details