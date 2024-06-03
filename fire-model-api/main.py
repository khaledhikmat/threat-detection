import random
import string
from dataclasses import dataclass
from typing import Dict

from fastapi import FastAPI, HTTPException, Response, status

app = FastAPI()

@dataclass
class Detection:
    id: str
    url: str

detections: Dict[str, Detection] = {}

letters = string.ascii_letters

# Generate some random detections
for i in range(10):
    id = str(random.randint(1, 100))
    url = ''.join(random.choice(letters) for k in range(10)) + '/' + ''.join(random.choice(letters) for k in range(5)) + '.jpg'
    detections[id] = Detection(id=id, url=url)

"""
    Return whether the API is running or not
"""
@app.get("/ping")
def ping() -> Response:
    return Response("Fire model API is running!!", status_code=status.HTTP_200_OK)

"""
    Detect fire properties in the file located at the given storage URL
"""
@app.post("/detections", response_model=Detection)
def detect_fire(item: Detection) -> Detection:
    # TODO: Implement fire detection logic here
    # For now, just return a random detection
    id = str(random.randint(1, 100))
    if id in detections:
        return detections[id]
    
    # Empty URL is an indication that the detection did not turn up anything
    return Detection(id="", url="")

