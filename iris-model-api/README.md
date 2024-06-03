This is a sample FAST API that calls upon a real nn model to predict. It works locally but there are some challenges:
- The Docker image it produces in huge i.e. > 6 GB! There must be some techniques to make it smaller. 
- Not sure how it handles concurrency. 

To create a virtual environment:

```bash
cd iris-model-api
python3 -m venv .venv
source .venv/bin/activate
```

To install pip packages:

```bash
pip install -r requirements.txt
pip list
```

To stop a virtual environment:

```bash
deactivate
```