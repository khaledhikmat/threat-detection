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

## DAPR

To run locallY using one terminal session:

```bash
make start-dapr
```
To stop a locallY running instance using another terminal session:

```bash
./stop-dapr
```

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
AWS_CHCKET_PREFIX="something-unique"
SQLLITE_FILE_PATH="<your-path>/db/clips.db" 
```


