#!/bin/bash

echo "Setting up ML environment..."

# Create required directories
mkdir -p data
mkdir -p models

# Check if Suicide_Detection.csv exists at the root level
if [ -f "../Suicide_Detection.csv" ]; then
    echo "Found Suicide_Detection.csv at root level, copying to data directory..."
    cp ../Suicide_Detection.csv data/
elif [ -f "Suicide_Detection.csv" ]; then
    echo "Found Suicide_Detection.csv in current directory, moving to data directory..."
    mv Suicide_Detection.csv data/
else
    echo "Warning: Suicide_Detection.csv not found. Please place it in the data directory manually."
fi

# Create .env file if it doesn't exist
if [ ! -f ".env" ]; then
    echo "Creating .env file from example..."
    cp .env.example .env
    echo "Please update the .env file with your Gemini API key."
fi

echo "Setup completed!" 