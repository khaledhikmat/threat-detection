import random
import torch
import numpy as np

x = torch.rand(5, 3)
print (x)

# You can use Google CO Lab to run this code
# google colab is a free jupyter notebook environment that requires no setup and runs entirely in the cloud.
# https://colab.research.google.com/notebooks/intro.ipynb
# Paid version is also available for more resources and GPU support.

# Resources:
# https://github.com/flatplanet/Pytorch-Tutorial-Youtube/tree/main
# https://pytorch.org/tutorials/beginner/deep_learning_60min_blitz.html
# pytorch.org
# Integration with Github

# Tensor is a multi-dimensional matrix containing elements of a single data type.
# Tensors are similar to NumPy’s ndarrays, with the addition being that Tensors can also be used on a GPU to accelerate computing.
# torch.Tensor

# Lists
lst1 = [1, 2, 3, 4, 5]
print(lst1)

lst2 = [[1, 2, 3, 4, 5], [1, 2, 3, 4, 5]]
print(lst2)

# Numpy
np1 = np.random.rand(3, 4)
print(np1)
np1.dtype

# Tensor
t_2d = torch.randn(3, 4)
print(t_2d)
t_2d.dtype

t_3d = torch.zeros(2, 3, 4)
print(t_3d)
t_3d.dtype

# Tensor Operations
my_torch = torch.arange(10)
print(my_torch)

# Reshape
my_torch_reshaped = my_torch.reshape(2, 5)
print(my_torch_reshaped)

# Reshape if we don't know the number of columns
my_torch_reshaped = my_torch.reshape(2, -1)
print(my_torch_reshaped)

# View is another reshape function
my_torch_reshaped = my_torch.view(2, 5)
print(my_torch_reshaped)

# Google the difference between reshape and view

# Updates are auto-reflected in the original tensor
my_torch[0] = 100

# Slices
my_torch = torch.arange(10)
print(my_torch)
# Grab a specific item
print(my_torch[0])
my_torch = my_torch.reshape(5, 2)
print(my_torch)
print(my_torch[:, 1])
print(my_torch[:, 1:])

# Math
t1 = torch.tensor([1, 2, 3, 4])
t2 = torch.tensor([5, 6, 7, 8])
print(t1 + t2)
torch.add(t1, t2) # Same as above

print(t1 - t2)
torch.sub(t1, t2) # Same as above

print(t1 * t2)
torch.mul(t1, t2) # Same as above

print(t1 / t2)
torch.div(t1, t2) # Same as above

# Inplace operations
t1.add_(t2)
print(t1)

# Matrix Multiplication
m1 = torch.randn(2, 2)
m2 = torch.randn(2, 2)
print(m1)
print(m2)
m3 = torch.mm(m1, m2)   

random_floats = [random.uniform(0, 10) for _ in range(4)]
mystery_iris = torch.tensor(random_floats)
print(mystery_iris)






