#!/bin/bash

# Volume Syncer Test Script
# This script tests the volume syncer API with a sample SSH sync request

API_URL="http://localhost:8080/api/1.0/sync"

echo "Testing Volume Syncer API..."

# Example SSH sync request with source path
curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "ssh",
      "details": {
        "host": "sshServer",
        "port": 22,
        "username": "pdm",
        "path": "/opt/data",
        "privateKey": "LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0K...replace-with-your-actual-base64-key..."
      }
    },
    "target": {
      "path": "/Users/bilgehan.nal/Desktop/test/cp"
    },
    "timeout": "30s"
  }'

echo ""
echo "Request sent!"
echo ""
echo "Note: Replace the privateKey with your actual base64-encoded private key"
