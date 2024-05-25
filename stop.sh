dapr stop -f . &&
lsof -i:8080 | grep main | awk '{print $2}' | xargs kill &&
lsof -i:8081 | grep model | awk '{print $2}' | xargs kill &&
lsof -i:8082 | grep model | awk '{print $2}' | xargs kill &&
lsof -i:8083 | grep alert | awk '{print $2}' | xargs kill &&
lsof -i:8084 | grep alert | awk '{print $2}' | xargs kill &&
lsof -i:8085 | grep alert | awk '{print $2}' | xargs kill &&
lsof -i:8086 | grep alert | awk '{print $2}' | xargs kill &&
lsof -i:8087 | grep media | awk '{print $2}' | xargs kill &&
lsof -i:8088 | grep media | awk '{print $2}' | xargs kill &&
lsof -i:8089 | grep media | awk '{print $2}' | xargs kill

