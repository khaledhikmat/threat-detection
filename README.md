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

## Dockerize

*Because the solution is a Go workspace and relies on a shared library, it is important to tag the shared lib and update the microservice modules.*

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
- In each microservice folder, perform the following to update the shared library:
    - `go get -u go get -u github.com/khaledhikmat/threat-detection-shared@v1.0.0`. Replace `v1.0.0` with your actual tag.
    - `go mod tidy`
- `make dockerize`. This buildsand dockerizes all microservices.
- `make push-2-hub`. This builds, dockerizes and pushes microservice Docker images to a public Docker repo i.e. Docker Hub.
- Merge and tag as above.

**Please note** that:
- Sometimes `make push-2-hub` times out! In this case, you have to execute the command one by one 😢.  
- The image names must be formatted this way: `<accountname>/<image-name>:tag`.
- The image architecture must be `linux/amd64`. So if you are on MacOS M1/M2 chip, you must instruct Docker to build using amd64 platform: `buildx build --platform linux/amd64`.

## Environment Variables

There are two runtime environments:
- `local`: used for laptop deployment. `dapr.yaml` is used to define all the microservices that must run.
- `higher`: used for Cloud deployment. Microservices are expected to run within Docker containers.  

There are two runtime modes:
- `dapr`: uses DAPR components and local Docker containers (such as Redis and Redis Streams) to provide functionality for storage (i.e. REDIS), pubsub (i.e. REDID Streams) and persistence (i.e. REDIS and SQLLite). 
- `aws`: uses AWS services to ptovide such as SQS, SNS and S3 to provide functionality for storage (i.e. S3), pubsub (i.e. SNS and SQS) and persistence (i.e. AWS OpenSearch and SQLLite).

**Please note** that:
- While running locally, `dapr.yml` is used to define microservices that must be launched regardless of whether the runtime mode is `dapr` or `aws`.
- Some env variables are specified in `dapr.yml` file while others (that expose account information) are stored in `.env` file. The `.env` file is added to `.gitgignore` so it is not pushed to source control.

The following are the required env variables for each microservice:

### Camera Stream Capturer

| VAR | DESC | DEFAULT |
| --- | --- | --- |
| `RUN_TIME_ENV` | some desc | `local` |
| `RUN_TIME_MODE` | some desc | `aws` |
| `AWS_ACCESS_KEY_ID` | some desc | `personal AWS account` |
| `AWS_SECRET_ACCESS_KEY` | some desc | `personal AWS account` |
| `AGENT_MODE` | some desc | `files` |
| `AGENT_RECORDINGS_FOLDER` | some desc | `./data/recordings` |
| `AGENT_SAMPLES_FOLDER` | some desc | `./data/samples` |
| `CAPTURER_MAX_CAMERAS` | some desc | `3` |

### Model Invoker

There can be several deployments of this Microservice so we can invoke all the models that we have (or will have):
- `weapon`
- `fire`
- `crowd`
- etc. 

The `AI_MODEL` specifies the type.

| VAR | DESC | DEFAULT |
| --- | --- | --- |
| `RUN_TIME_ENV` | some desc | `local` |
| `RUN_TIME_MODE` | some desc | `aws` |
| `AWS_ACCESS_KEY_ID` | some desc | `personal AWS account` |
| `AWS_SECRET_ACCESS_KEY` | some desc | `personal AWS account` |
| `AI_MODEL` | some desc | `weapon` |

### Media Indexer

There can be several deployments of this Microservice so we can index to the destinations that we need:
- `database`
- `elastic`
- etc. 

The `MEDIA_INDEXER_TYPE` specifies the media indexer type while the `INDEXER_TYPE` specifies the actual implementation.

| VAR | DESC | DEFAULT |
| --- | --- | --- |
| `RUN_TIME_ENV` | some desc | `local` |
| `RUN_TIME_MODE` | some desc | `aws` |
| `AWS_ACCESS_KEY_ID` | some desc | `personal AWS account` |
| `AWS_SECRET_ACCESS_KEY` | some desc | `personal AWS account` |
| `SQLLITE_FILE_PATH` | some desc | `/Users/khaled/github/threat-detection/db/clips.db` |
| `OPEN_SEARCH_DOMAIN_ENDPOINT` | some desc | `https://<your-domain>.aos.us-east-2.on.aws` |
| `OPEN_SEARCH_INDEX_NAME` | some desc | `<your-index-name>` |
| `OPEN_SEARCH_USERNAME` | some desc | `<your-master-username>` |
| `OPEN_SEARCH_PASSWORD` | some desc | `<your-master-password>` |
| `MEDIA_INDEXER_TYPE` | some desc | `elastic` |
| `INDEXER_TYPE` | som desc | `opensearch` |

### Media API

| VAR | DESC | DEFAULT |
| --- | --- | --- |
| `RUN_TIME_ENV` | some desc | `local` |
| `RUN_TIME_MODE` | some desc | `aws` |
| `AWS_ACCESS_KEY_ID` | some desc | `personal AWS account` |
| `AWS_SECRET_ACCESS_KEY` | some desc | `personal AWS account` |
| `SQLLITE_FILE_PATH` | some desc | `/Users/khaled/github/threat-detection/db/clips.db` |
| `OPEN_SEARCH_DOMAIN_ENDPOINT` | some desc | `https://<your-domain>.aos.us-east-2.on.aws` |
| `OPEN_SEARCH_INDEX_NAME` | some desc | `<your-index-name>` |
| `OPEN_SEARCH_USERNAME` | some desc | `<your-master-username>` |
| `OPEN_SEARCH_PASSWORD` | some desc | `<your-master-password>` |
| `MEDIA_INDEXER_TYPE` | some desc | `elastic` |
| `INDEXER_TYPE` | som desc | `opensearch` |
| `APP_PORT` | som desc | `8080` |

### Alert API

There can be several deployments of this Microservice so we can invoke all the upstream application we need to notify:
- `ccure`
- `slack`
- `snow`
- `perspective`
- etc. 

The `ALERT_TYPE` specifies the type.

| VAR | DESC | DEFAULT |
| --- | --- | --- |
| `RUN_TIME_ENV` | some desc | `local` |
| `RUN_TIME_MODE` | some desc | `aws` |
| `AWS_ACCESS_KEY_ID` | some desc | `personal AWS account` |
| `AWS_SECRET_ACCESS_KEY` | some desc | `personal AWS account` |
| `ALERT_TYPE` | some desc | `snow` |

## Deployment

AWS is the only Cloud vendor we are considering at this time for this solution. The below show a manual deployment. But we need to work on CLI pipeline to push these resources to AWS.

### Storage

Buckets in S3 are created automatically as needed. There is a bucket for each camera.

### SQS and SNS

In local mode, SNS topics and SQS queues are automatically created if they do not exist. So when we deploy to AWS, we expect these resources to be created. 

### OpenSearch Service

A domain (cluster + index) needs to be created ahead of deployment. Some of the env variables below rely on OpenSearch being available.

### Elastic Container Service on FARGATE

Cluster: `kh-td-poc-ecs-on-fargate`

#### Task Definitions

The following are the task definitions required to run the solution:

- `kh-td-poc-camera-stream-capturer`:

```json
{
    "containerDefinitions": [
        {
            "name": "camera-stream-capturer",
            "image": "khaledhikmat/threat-detection-camera-stream-capturer:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AGENT_MODE",
                    "value": "files"
                },
                {
                    "name": "CAPTURER_MAX_CAMERAS",
                    "value": "3"
                },
                {
                    "name": "AGENT_SAMPLES_FOLDER",
                    "value": "./data/samples"
                },
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "AGENT_RECORDINGS_FOLDER",
                    "value": "./data/recordings"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-camera-stream-capturer",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-camera-stream-capturer",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-weapon-model-invoker`:

```json
{
    "containerDefinitions": [
        {
            "name": "weapon-model-invoker",
            "image": "khaledhikmat/threat-detection-model-invoker:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "AI_MODEL",
                    "value": "weapon"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-weapon-model-invoker",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-weapon-model-invoker",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-fire-model-invoker`:

```json
{
    "containerDefinitions": [
        {
            "name": "fire-model-invoker",
            "image": "khaledhikmat/threat-detection-model-invoker:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "AI_MODEL",
                    "value": "fire"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-fire-model-invoker",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-fire-model-invoker",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-ccure-alert-notifier`:

```json
{
    "containerDefinitions": [
        {
            "name": "ccure-alert-notifier",
            "image": "khaledhikmat/threat-detection-alert-notifier:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "ALERT_TYPE",
                    "value": "ccure"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-ccure-alert-notifier",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-ccure-alert-notifier",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-snow-alert-notifier`:

```json
{
    "containerDefinitions": [
        {
            "name": "snow-alert-notifier",
            "image": "khaledhikmat/threat-detection-alert-notifier:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "ALERT_TYPE",
                    "value": "snow"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-snow-alert-notifier",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-snow-alert-notifier",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-pers-alert-notifier`:

```json
{
    "containerDefinitions": [
        {
            "name": "pers-alert-notifier",
            "image": "khaledhikmat/threat-detection-alert-notifier:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "ALERT_TYPE",
                    "value": "pers"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-pers-alert-notifier",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-pers-alert-notifier",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-slack-alert-notifier`:

```json
{
    "containerDefinitions": [
        {
            "name": "slack-alert-notifier",
            "image": "khaledhikmat/threat-detection-alert-notifier:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "ALERT_TYPE",
                    "value": "slack"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-slack-alert-notifier",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-slack-alert-notifier",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-elastic-media-indexer`:

```json
{
    "containerDefinitions": [
        {
            "name": "elastic-media-indexer",
            "image": "khaledhikmat/threat-detection-media-indexer:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "OPEN_SEARCH_DOMAIN_ENDPOINT",
                    "value": "<your-key>"
                },
                {
                    "name": "OPEN_SEARCH_INDEX_NAME",
                    "value": "kh-td-opc-open-search"
                },
                {
                    "name": "OPEN_SEARCH_USERNAME",
                    "value": "<your-master-username>"
                },
                {
                    "name": "OPEN_SEARCH_PASSWORD",
                    "value": "<your-master-password>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "MEDIA_INDEXER_TYPE",
                    "value": "elastic"
                },
                {
                    "name": "INDEXER_TYPE",
                    "value": "opensearch"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-elastic-media-indexer",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-elastic-media-indexer",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

- `kh-td-poc-media-api`:

```json
{
    "containerDefinitions": [
        {
            "name": "media-api",
            "image": "khaledhikmat/threat-detection-media-api:latest",
            "cpu": 0,
            "portMappings": [],
            "essential": true,
            "environment": [
                {
                    "name": "AWS_ACCESS_KEY_ID",
                    "value": "<your-key>"
                },
                {
                    "name": "AWS_SECRET_ACCESS_KEY",
                    "value": "<your-key>"
                },
                {
                    "name": "OPEN_SEARCH_DOMAIN_ENDPOINT",
                    "value": "<your-key>"
                },
                {
                    "name": "OPEN_SEARCH_INDEX_NAME",
                    "value": "kh-td-opc-open-search"
                },
                {
                    "name": "OPEN_SEARCH_USERNAME",
                    "value": "<your-master-username>"
                },
                {
                    "name": "OPEN_SEARCH_PASSWORD",
                    "value": "<your-master-password>"
                },
                {
                    "name": "RUN_TIME_MODE",
                    "value": "aws"
                },
                {
                    "name": "RUN_TIME_ENV",
                    "value": "REVIEW"
                },
                {
                    "name": "MEDIA_INDEXER_TYPE",
                    "value": "elastic"
                },
                {
                    "name": "INDEXER_TYPE",
                    "value": "opensearch"
                }
            ],
            "environmentFiles": [],
            "mountPoints": [],
            "volumesFrom": [],
            "ulimits": [],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-create-group": "true",
                    "awslogs-group": "/ecs/kh-td-poc-media-api",
                    "awslogs-region": "us-east-2",
                    "awslogs-stream-prefix": "ecs"
                },
                "secretOptions": []
            },
            "systemControls": []
        }
    ],
    "family": "kh-td-poc-media-api",
    "taskRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "executionRoleArn": "arn:aws:iam::997763366404:role/ecsTaskExecutionRole",
    "networkMode": "awsvpc",
    "placementConstraints": [],
    "requiresCompatibilities": [
        "FARGATE"
    ],
    "cpu": "1024",
    "memory": "3072",
    "runtimePlatform": {
        "cpuArchitecture": "X86_64",
        "operatingSystemFamily": "LINUX"
    },
    "tags": []
}
```

#### Services

The following are the services to run the solution:

- `kh-td-poc-ccure-alert-notifier`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-snow-alert-notifier`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-pers-alert-notifier`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-slack-alert-notifier`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-elastic-media-indexer`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-elastic-media-api`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-weapon-model-invoker`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-fire-model-invoker`: 
    - desired tasks 1
    - Public IP
- `kh-td-poc-camera-stream-capturer`: 
    - desired tasks 1: of course, we can add additional tasks to handle camera streaming load. 
    - Public IP

**Please note** that while deploying services, we ran into some issues:
- Service errors can be located from the service -> events and click on the task to see failure reasons.
- Pull image errors cannot be located in logs because the log group has not been started yet!
- Locting the task errors from the service events revealed that pulling the Docker image from Docker Hub was timing out: `AWS ECS on Fargate failed to do request: Head "https://registry-1.docker.io/v2/my-styff: dial tcp 54.236.113.205:443: i/o timeout`.  
- After sleuthing around, found out this [Stack Overflow issue](https://stackoverflow.com/questions/77458662/aws-eks-on-fargate-pull-image-results-in-timeout) and this [Github issue](https://github.com/aws/amazon-ecs-agent/issues/2061).
- The security group in my case had the inbound and outbound wide open. 
- The Github issue referenced another [AWS documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_cannot_pull_image.html) which basically said to make the task have a public IP address. This seemed to have fixed the Docker pull problem. This is why I changed all the service configuration above to have a public IP address although it is not needed.
- The next problem is that the role that it auto-created to execute the task i.e. `ecsTaskExecutionRole` needs additional permissions to be able to create logs: `ResourceInitializationError: failed to validate logger args: create stream has been retried 1 times: failed to create Cloudwatch log group: AccessDeniedException: User: arn:aws:sts::997763366404:assumed-role/ecsTaskExecutionRole/d7795e52a4434edf80f4923f3b836d9f is not authorized to perform: logs:CreateLogGroup on resource: arn:aws:logs:us-east-2:997763366404:log-group:/ecs/kh-td-poc-ccure-alert-notifier:log-stream: because no identity-based policy allows the logs:CreateLogGroup action status code: 400, request id: 8d35f57a-a92f-4784-9922-ebb27eba0504 : exit status 1`.
- For now, I attached admin permissions to that role from the IAM console. This seems to have fixed the issue.
- I could not find a good way to restart a service that failed. I have been reducing the task count to 0, deleting the service and recreating it!! 
- It is probably better to delay the `kh-td-poc-camera-stream-capturer` service to the end. 

#### Deployment Notes

Once we got all the tasks running, it was a joy to see how they run together. A few notes:
- To get the task logs, we can do it from the service or from the task.
- If desired, we can stop tasks in one of two ways:
    - Go to an individual task and stop it. This will force the service to re-create another one because the service has a desired state of 1.
    - Go to the service, drain/set the desired tasks to 0 and update the service. This should not restart new tasks.
- Need to see how to create new revisions when the images get updated.
- Service Health and metrics is quite useful. It actually shows that we would be stressing the CPU and memory on a single instance.
 


