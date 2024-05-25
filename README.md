This project is an experimentation to create a simplified threat detection solution.

```bash
go work init ./shared ./camera-stream-capturer 
```

To add new modules:

```bash
go work edit -use ./alert-notifier 
```

To drop modules:

```bash
go work edit -dropuse ./alert-notifier 
```

## Diagrid

Generate an API Key

```bash
diagrid apikeys create --name mykey --role cra.diagrid:admin
```

Use the generated API Key when HSS remoted:

```bash
diagrid login --api-key <mykey>
```

Diagrid needs a declarative subscription to properly route pub/sub events to consumers:

```bash
diagrid subscription create recordings-pubsub --connection threat-detection-pubsub  --topic recordings-topic --route /recordings-topic --scopes model-invoker
```

```bash
diagrid subscription get recordings-pubsub --project eagle-threat-detection
```

```bash
diagrid subscription delete recordings-pubsub
```

To run locallY using one terminal session:

```bash
make start-diagrid
```
To stop a locallY running instance using another terminal session:

```bash
./stop-diagrid
```

**Please note** I am no longer supporting Diagrid.

## DAPR

To run locallY using one terminal session:

```bash
make start
```
To stop a locallY running instance using another terminal session:

```bash
./stop
```

**Please note** that I am using DAPR to orchestrate locally. So instead of using something like Docker compose (which requires that I Dockerize everything), DAPR provides an easy way to start all Microservices. This works even if the runtime mode is set to use `aws`.

## Redis

Start Local REDIS container:

```bash
docker exec -it dapr_redis redis-cli
```

## SQLLite

```bash
brew install sqlite3
brew install sqlite-utils
```

[https://earthly.dev/blog/golang-sqlite/](https://earthly.dev/blog/golang-sqlite/)
[https://www.allhandsontech.com/programming/golang/how-to-use-sqlite-with-go/](https://www.allhandsontech.com/programming/golang/how-to-use-sqlite-with-go/)

## Cleanup

To cleanup all resources in `Redis`:

```bash
FLUSHALL
```

To cleanup all resources in `SQLLite`:

```bash
cd <project-root>/db
rm clips.db
```

To cleanup all resources in `AWS S3`, login to the console portal and cleanup bucket items ma manually.

## Environment Variables

These should go in the `.env` file:

```bash
AWS_ACCESS_KEY_ID="<your-key>"
AWS_SECRET_ACCESS_KEY="<your-key>"
AWS_BUCKET_PREFIX="something-unique"
SQLLITE_FILE_PATH="<your-path>/db/clips.db" 
```

## Dockerize

- Merge and tag shared lib (`threat-detection-shared`) i.e. `v1.0.0`:
    - Assuming we have a working branch i.e. `my-branch`
    - `git add --all`
    - `git commit -am "Major stuff..."`
    - `git push`
    - `git checkout main`
    - `git merge my-branch`
    - `git tag -a v1.0.0 -m "my great work"`
    - `git tag` to make sure is is created.
    - `git push --tags` to push tags to Github.
- In all microservice folders:
    - `go get -u go get -u github.com/khaledhikmat/threat-detection-shared@v1.0.0`. Replace `v1.0.0` with your actual tag.
    - `go mod tidy`
- `make build`  
- `make dockerize`  
- `make push-2-hub`. This pushes to a public Docker repo i.e. Docker Hub.
- Merge and tag as above.

** Please note** that:
- I have noticed that sometimes that `make push-2-hub` times out. In this case, you have to execute the command one by one 😢.  
- The image names must be formatted this way: `<accountname>/<image-name>:tag`.
- The image architecture must be `linux/amd64`. So if you are on MacOS M1/M2 chip, you must instruct Docker to build using amd64 platform: `buildx build --platform linux/amd64`.

