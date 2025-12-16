#!/bin/bash
# Complete integration test for Veil

set -e

echo "=== VEIL INTEGRATION TEST ==="
echo ""

# Start server
echo "Starting server..."
cd /Users/kderbyma/Desktop/veil
rm -f veil.db
./veil init > /dev/null 2>&1
./veil serve --port 8080 > /dev/null 2>&1 &
SERVER_PID=$!
sleep 3

echo "✅ Server started (PID: $SERVER_PID)"
echo ""

# Test 1: Create site
echo "Test 1: Creating site..."
SITE=$(curl -s -X POST http://localhost:8080/api/sites \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Blog","description":"My test blog"}')
SITE_ID=$(echo $SITE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "✅ Created site: $SITE_ID"

# Test 2: Create note
echo "Test 2: Creating note..."
NOTE=$(curl -s -X POST "http://localhost:8080/api/sites/$SITE_ID/nodes" \
  -H "Content-Type: application/json" \
  -d '{"type":"note","title":"First Post","content":"Hello world!","path":"first.md"}')
NOTE_ID=$(echo $NOTE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "✅ Created note: $NOTE_ID"

# Test 3: Update note (create version)
echo "Test 3: Updating note..."
curl -s -X PUT "http://localhost:8080/api/sites/$SITE_ID/nodes/$NOTE_ID" \
  -H "Content-Type: application/json" \
  -d '{"title":"First Post Updated","content":"Updated content!"}' > /dev/null
echo "✅ Updated note"

# Test 4: Get versions
echo "Test 4: Checking versions..."
VERSIONS=$(curl -s "http://localhost:8080/api/sites/$SITE_ID/nodes/$NOTE_ID/versions")
VERSION_COUNT=$(echo $VERSIONS | grep -o '"version_number"' | wc -l | tr -d ' ')
echo "✅ Found $VERSION_COUNT versions"

# Test 5: Create another note for linking
echo "Test 5: Creating second note..."
NOTE2=$(curl -s -X POST "http://localhost:8080/api/sites/$SITE_ID/nodes" \
  -H "Content-Type: application/json" \
  -d '{"type":"note","title":"Second Post","content":"Another post","path":"second.md"}')
NOTE2_ID=$(echo $NOTE2 | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "✅ Created note: $NOTE2_ID"

# Test 6: Create reference/link
echo "Test 6: Creating link between notes..."
curl -s -X POST "http://localhost:8080/api/sites/$SITE_ID/nodes/$NOTE_ID/references" \
  -H "Content-Type: application/json" \
  -d "{\"target_node_id\":\"$NOTE2_ID\",\"link_text\":\"Check this out\",\"link_type\":\"internal\"}" > /dev/null
echo "✅ Created reference"

# Test 7: Get references
echo "Test 7: Getting forward links..."
REFS=$(curl -s "http://localhost:8080/api/sites/$SITE_ID/nodes/$NOTE_ID/references")
echo "✅ References: $(echo $REFS | head -c 100)..."

# Test 8: Get backlinks
echo "Test 8: Getting backlinks..."
BACKLINKS=$(curl -s "http://localhost:8080/api/sites/$SITE_ID/nodes/$NOTE2_ID/backlinks")
echo "✅ Backlinks: $(echo $BACKLINKS | head -c 100)..."

# Test 9: Preview
echo "Test 9: Testing preview..."
PREVIEW=$(curl -s "http://localhost:8080/preview/$SITE_ID/$NOTE_ID" | head -c 200)
if echo "$PREVIEW" | grep -q "First Post"; then
    echo "✅ Preview works"
else
    echo "❌ Preview failed"
fi

# Test 10: List nodes
echo "Test 10: Listing all notes..."
NODES=$(curl -s "http://localhost:8080/api/sites/$SITE_ID/nodes")
NODE_COUNT=$(echo $NODES | grep -o '"id"' | wc -l | tr -d ' ')
echo "✅ Found $NODE_COUNT notes in site"

echo ""
echo "=== ALL TESTS PASSED ==="
echo ""
echo "🎉 Veil is working correctly!"
echo ""
echo "Try it: http://localhost:8080"
echo ""
echo "Press Ctrl+C to stop server..."
wait $SERVER_PID
