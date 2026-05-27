#!/bin/bash

# Test script for purchase functionality
# This script tests the purchase API endpoints

BASE_URL="http://localhost:8080"
USERNAME="admin"
PASSWORD="admin123"

echo "🧪 Testing Purchase System"
echo "=========================="
echo ""

# Step 1: Login
echo "1️⃣  Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ Login failed!"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
fi

echo "✅ Login successful"
echo "Token: ${TOKEN:0:50}..."
echo ""

# Step 2: Get Products
echo "2️⃣  Fetching products..."
PRODUCTS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/products" \
  -H "Authorization: Bearer $TOKEN")

echo "Products:"
echo "$PRODUCTS_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$PRODUCTS_RESPONSE"
echo ""

# Step 3: Make a Purchase
echo "3️⃣  Making a purchase..."
PURCHASE_DATA='{
  "user_id": "admin",
  "items": [
    {
      "product_id": 1,
      "quantity": 1
    },
    {
      "product_id": 2,
      "quantity": 2
    }
  ]
}'

PURCHASE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/purchase" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$PURCHASE_DATA")

echo "Purchase Response:"
echo "$PURCHASE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$PURCHASE_RESPONSE"
echo ""

# Step 4: Get Orders
echo "4️⃣  Fetching orders..."
ORDERS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/orders?user_id=admin" \
  -H "Authorization: Bearer $TOKEN")

echo "Orders:"
echo "$ORDERS_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$ORDERS_RESPONSE"
echo ""

# Step 5: Check Products Again (to see stock updated)
echo "5️⃣  Checking products again (stock should be updated)..."
PRODUCTS_RESPONSE2=$(curl -s -X GET "$BASE_URL/api/products" \
  -H "Authorization: Bearer $TOKEN")

echo "Updated Products:"
echo "$PRODUCTS_RESPONSE2" | python3 -m json.tool 2>/dev/null || echo "$PRODUCTS_RESPONSE2"
echo ""

echo "✅ Test completed!"
