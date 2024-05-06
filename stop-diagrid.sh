unset THREAT_DETECTION_MODE && diagrid dev stop -f dev-eagle-threat-detection.yaml
(lsof -i:8080 | grep main) | awk '{print $2}' | xargs kill &&
(lsof -i:8081 | grep main) | awk '{print $2}' | xargs kill &&
(lsof -i:8082 | grep main) | awk '{print $2}' | xargs kill &&
(lsof -i:8083 | grep main) | awk '{print $2}' | xargs kill &&
(lsof -i:3000 | grep main) | awk '{print $2}' | xargs kill

