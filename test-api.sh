#!/bin/bash

# Example script for testing the volume syncer API

API_URL="http://localhost:8080"

echo "=== Volume Syncer API Test ==="
echo

# Test health endpoint
echo "1. Testing health endpoint..."
curl -s "$API_URL/health" | jq .
echo

# Example sync request (you need to replace with actual SSH details)
echo "2. Example sync request (update with your SSH details):"
echo "curl -X POST $API_URL/api/1.0/sync \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{"
echo "    \"source\": {"
echo "      \"type\": \"ssh\","
echo "      \"details\": {"
echo "        \"host\": \"your-ssh-server.com\","
echo "        \"port\": 22,"
echo "        \"username\": \"your-username\","
echo "        \"privateKey\": \"LS0tLS1CRUdJTi...\""
echo "      }"
echo "    },"
echo "    \"target\": {"
echo "      \"path\": \"/mnt/shared-volume\""
echo "    },"
echo "    \"timeout\": \"60s\""
echo "  }'"
echo

# Test invalid request
echo "3. Testing invalid request (should return 400)..."
curl -s -X POST "$API_URL/api/1.0/sync" \
  -H "Content-Type: application/json" \
  -d '{"invalid": "request"}' | jq .
echo

echo "=== Test completed ==="
echo
echo "To generate a base64 private key:"
echo "base64 -w 0 ~/.ssh/id_rsa"
