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

Start Local REDIS container:

```bash
docker exec -it dapr_redis redis-cli
```

To run locallY using one terminal session:

```bash
make start-dapr
```
To stop a locallY running instance using another terminal session:

```bash
./stop-dapr
```


