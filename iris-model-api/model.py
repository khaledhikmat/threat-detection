import torch
import torch.nn as nn
import torch.nn.functional as F

import pandas as pd
# import matplotlib.pyplot as plt
# %matplotlib inline
from sklearn.model_selection import train_test_split

# Create a Model Class that inherits nn.Module
class Model(nn.Module):
  # Input layer (4 features of the flower) -->
  # Hidden Layer1 (number of neurons) -->
  # H2 (n) -->
  # output (3 classes of iris flowers)
  def __init__(self, in_features=4, h1=8, h2=9, out_features=3):
    super().__init__() # instantiate our nn.Module
    self.fc1 = nn.Linear(in_features, h1)
    self.fc2 = nn.Linear(h1, h2)
    self.out = nn.Linear(h2, out_features)

  def forward(self, x):
    x = F.relu(self.fc1(x))
    x = F.relu(self.fc2(x))
    x = self.out(x)

    return x

# Pick a manual seed for randomization
torch.manual_seed(41)
# Create an instance of model
model = Model()

# Load the iris dataset
url = 'iris-data-set.csv'
my_df = pd.read_csv(url)
print(my_df.head())

# Change last column from strings to integers
my_df['variety'] = my_df['variety'].replace('Setosa', 0.0)
my_df['variety'] = my_df['variety'].replace('Versicolor', 1.0)
my_df['variety'] = my_df['variety'].replace('Virginica', 2.0)
print(my_df.head())

# Train Test Split!  Set X, y
X = my_df.drop('variety', axis=1)
y = my_df['variety']  

# Convert these to numpy arrays
X = X.values
y = y.values

# Use 20% of the data for testing
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=33)

# Convert X features to float tensors
X_train = torch.FloatTensor(X_train)
X_test = torch.FloatTensor(X_test)

# Convert y to LongTensor
y_train = torch.LongTensor(y_train)
y_test = torch.LongTensor(y_test)

# Loss Function
criterion = nn.CrossEntropyLoss()
# Choose Adam Optimizer, lr = learning rate (if error doesn't go down after a bunch of iterations (epochs), lower our learning rate)
optimizer = torch.optim.Adam(model.parameters(), lr=0.01)

# Train our model!
# Epochs? (one run thru all the training data in our network)
epochs = 100
losses = []
for i in range(epochs):
  # Go forward and get a prediction
  y_pred = model.forward(X_train) # Get predicted results

  # Measure the loss/error, gonna be high at first
  loss = criterion(y_pred, y_train) # predicted values vs the y_train

  # Keep Track of our losses
  losses.append(loss.detach().numpy())

  # print every 10 epoch
  if i % 10 == 0:
    print(f'Epoch: {i} and loss: {loss}')

  # Do some back propagation: take the error rate of forward propagation and feed it back
  # thru the network to fine tune the weights
  optimizer.zero_grad()
  loss.backward()
  optimizer.step()

# Graph it out!
# plt.plot(range(epochs), losses)
# plt.ylabel("loss/error")
# plt.xlabel('Epoch')

# Evaluate Model on Test Data Set (validate model on test set)
with torch.no_grad():  # Basically turn off back propogation
  y_eval = model.forward(X_test) # X_test are features from our test set, y_eval will be predictions
  loss = criterion(y_eval, y_test) # Find the loss or error

print(f'Loss: {loss}')

# Check the accuracy of the model
correct = 0
with torch.no_grad():
  for i, data in enumerate(X_test):
    y_val = model.forward(data)

    if y_test[i] == 0:
      x = "Setosa"
    elif y_test[i] == 1:
      x = 'Versicolor'
    else:
      x = 'Virginica'


    # Will tell us what type of flower class our network thinks it is
    print(f'{i+1}.)  {str(y_val)} \t {y_test[i]} \t {y_val.argmax().item()}')

    # Correct or not
    if y_val.argmax().item() == y_test[i]:
      correct +=1

print(f'We got {correct} correct!')


# Save the NN model
torch.save(model.state_dict(), 'iris_model.pth')

# Load the model
new_model = Model()
new_model.load_state_dict(torch.load('iris_model.pth'))

# Make sure it loaded correctly
new_model.eval()

# Make a prediction
mystery_iris = torch.tensor([5.6, 3.7, 2.2, 0.5])  # Setosa
print(new_model(mystery_iris).argmax().item())  # 0

mystery_iris = torch.tensor([6.7, 3.3, 5.7, 2.1])  # Virginica
print(new_model(mystery_iris).argmax().item())  # 2

# Path: prediction-model/main.py 




