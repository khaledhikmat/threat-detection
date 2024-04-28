BUILD_DIR="./build"
DIST_DIR="./dist"

clean_build:
	if [ -d "${BUILD_DIR}" ]; then rm -r ${BUILD_DIR}; fi

clean_dist:
	if [ -d "${DIST_DIR}" ]; then rm -r ${DIST_DIR}; fi; mkdir ${DIST_DIR}

test:
	echo "Invoking test cases..."

build: clean_dist clean_build test
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-camera-stream-capturer" ./camera-stream-capturer/main.go
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-ccure-invoker" ./ccure-invoker/main.go
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-clip-indexer" ./clip-indexer/main.go
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-model-invoker" ./model-invoker/main.go
	GOOS='linux' GOARCH='amd64' GO111MODULE='on' go build -o "${BUILD_DIR}/threat-detection-api" ./api/main.go

dockerize: clean_dist clean_build test build
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-camera-stream-capturer:latest ./camera-stream-capturer -f ./camera-stream-capturer/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-ccure-invoker:latest ./ccure-invoker -f ./ccure-invoker/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-clip-indexer:latest ./clip-indexer -f ./clip-indexer/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-model-invoker:latest ./model-invoker -f ./model-invoker/Dockerfile
	docker buildx build --platform linux/amd64 -t khaledhikmat/threat-detection-api:latest ./api-f ./api/Dockerfile

start: clean_dist clean_build test build
	dapr run -f .

list: 
	dapr list

stop: 
	./stop.sh
	# For some reason, the awk command is not working in makefile
	# Ignore errors by placing `-` at the beginning of the line	
	# -dapr stop -f .
	# -(lsof -i:8080 | grep main) | awk '{print $2}' | xargs kill
	# -(lsof -i:8081 | grep main) | awk '{print $2}' | xargs kill
	# -(lsof -i:8082 | grep main) | awk '{print $2}' | xargs kill
	# -(lsof -i:8083 | grep main) | awk '{print $2}' | xargs kill
	# -(lsof -i:3000 | grep main) | awk '{print $2}' | xargs kill