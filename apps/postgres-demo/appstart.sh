#!/bin/bash

# PostgreSQL Demo Application Runner
# This script runs the PostgreSQL demo application

echo "Starting PostgreSQL Demo Application..."
echo ""

# Check if PostgreSQL is running (optional check)
if command -v pg_isready &> /dev/null; then
    if pg_isready -h localhost -p 5432 &> /dev/null; then
        echo "✅ PostgreSQL is running on localhost:5432"
    else
        echo "⚠️  Warning: PostgreSQL may not be running on localhost:5432"
        echo "   Make sure PostgreSQL is running and accessible"
    fi
    echo ""
fi

# Check if application.properties exists
if [ ! -f "application.properties" ]; then
    echo "❌ Error: application.properties not found"
    echo "   Please create application.properties with your database configuration"
    exit 1
fi

# Run the application
echo "Running application..."
go run .
