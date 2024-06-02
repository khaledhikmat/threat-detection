BUILD_DIR="./build"
DIST_DIR="./dist"

clean_build:
	if [ -d "${BUILD_DIR}" ]; then rm -r ${BUILD_DIR}; fi

clean_dist:
	if [ -d "${DIST_DIR}" ]; then rm -r ${DIST_DIR}; fi; mkdir ${DIST_DIR}

test:
	echo "Invoking test cases..."

build: clean_dist clean_build test
	# For now, we are only building the camera-stream-capturer using the ARM64 architecture
	# This is because the camera-stream-capturer is the only component that is dependent on CGO and C libs
	CGO_ENABLED=1 GOOS='darwin' GOARCH='arm64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-camera-stream-capturer" ./camera-stream-capturer/.
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-model-invoker" ./model-invoker/.
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-alert-notifier" ./alert-notifier/.
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-media-indexer" ./media-indexer/.
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-media-api" ./media-api/.

dockerize: clean_dist clean_build test build
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-camera-stream-capturer:latest ./camera-stream-capturer -f ./camera-stream-capturer/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-model-invoker:latest ./model-invoker -f ./model-invoker/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-alert-notifier:latest ./alert-notifier -f ./alert-notifier/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-media-indexer:latest ./media-indexer -f ./media-indexer/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-media-api:latest ./media-api -f ./media-api/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-weapon-model-api:latest ./weapon-model-api -f ./weapon-model-api/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-fire-model-api:latest ./fire-model-api -f ./fire-model-api/Dockerfile

push-2-hub: clean_dist clean_build test build dockerize
	docker login
	docker push khaledhikmat/threat-detection-camera-stream-capturer:latest
	docker push khaledhikmat/threat-detection-model-invoker:latest	
	docker push khaledhikmat/threat-detection-alert-notifier:latest
	docker push khaledhikmat/threat-detection-media-indexer:latest
	docker push khaledhikmat/threat-detection-media-api:latest
	docker push khaledhikmat/threat-detection-weapon-model-api:latest
	docker push khaledhikmat/threat-detection-fire-model-api:latest

start: clean_dist clean_build test
	dapr run -f .

start-single: clean_dist clean_build test
	dapr run -f ./dapr-single.yaml

list: 
	dapr list

stop: 
	#./stop.sh
	dapr stop -f . && (lsof -i:8080 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:8081 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:8082 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:8083 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:3000 | grep main) | awk '{print $2}' | xargs kill

stop-single: 
	#./stop-single.sh
	dapr stop -f ./dapr-single.yaml && (lsof -i:8080 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:8081 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:8082 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:8083 | grep main) | awk '{print $2}' | xargs kill && (lsof -i:3000 | grep main) | awk '{print $2}' | xargs kill

run-aws-collector:
	docker run -d --rm -p 4317:4317 -p 55679:55679 -p 8889:8888 \
			-e "AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID)" \
			-e "AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY)" \
			-e "AWS_REGION=$(AWS_REGION)" \
            -v "${PWD}/telemetry/aws-collector-config.yaml":/otel-local-config.yaml \
            --name awscollector public.ecr.aws/aws-observability/aws-otel-collector:latest \
            --config otel-local-config.yaml; \

stop-aws-collector:
	docker stop awscollector

run-model-apis:
	docker run -d --rm -p 5001:5001 \
		    --platform linux/amd64 \
            --name weapon-model-api khaledhikmat/threat-detection-weapon-model-api:latest; \
	docker run -d --rm -p 5002:5002 \
		    --platform linux/amd64 \
            --name fire-model-api khaledhikmat/threat-detection-fire-model-api:latest; \

stop-model-apis:
	docker stop weapon-model-api
	docker stop fire-model-api
