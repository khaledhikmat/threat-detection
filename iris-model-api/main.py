import random
import string
from dataclasses import dataclass
from typing import Dict

import torch
import torch.nn as nn
import torch.nn.functional as F

import pandas as pd
# import matplotlib.pyplot as plt
# %matplotlib inline
from sklearn.model_selection import train_test_split

from fastapi import FastAPI, HTTPException, Response, status
from model import Model

# Load the iris model
iris_model = Model()
iris_model.load_state_dict(torch.load('iris_model.pth'))

# Make sure it loaded correctly
iris_model.eval()

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
    return Response("Iris model API is running!!", status_code=status.HTTP_200_OK)

"""
    Detect iris properties in the file located at the given storage URL
"""
@app.post("/detections", response_model=Detection)
def detect_iris(item: Detection) -> Detection:
    # Make an iris prediction using random data
    random_floats = [random.uniform(0, 10) for _ in range(4)]
    mystery_iris = torch.tensor(random_floats)
    if iris_model(mystery_iris).argmax().item() == 0:
        return Detection(id="1", url="https://picsum.photos/200/300")
    
    # Empty URL is an indication that the detection did not turn up anything
    return Detection(id="", url="")

