# Hướng dẫn Test Hệ thống Mua hàng

## Cách 1: Test qua Web UI (Khuyên dùng)

### Bước 1: Khởi động ứng dụng

```bash
cd apps/postgres-demo
go run main.go
```

Ứng dụng sẽ:
- Kết nối với PostgreSQL
- Tạo các bảng (products, orders, order_items)
- Seed dữ liệu sản phẩm mẫu
- Khởi động web server trên port 8080

### Bước 2: Mở trình duyệt

1. Truy cập: http://localhost:8080
2. Đăng nhập với:
   - **Username**: `admin`
   - **Password**: `admin123`

### Bước 3: Test mua hàng

1. **Xem sản phẩm**: Trang sẽ hiển thị danh sách sản phẩm với giá và số lượng tồn kho
2. **Thêm vào giỏ hàng**:
   - Chọn số lượng cho mỗi sản phẩm
   - Click nút "Add"
   - Sản phẩm sẽ xuất hiện trong giỏ hàng
3. **Hoàn tất mua hàng**:
   - Xem lại giỏ hàng và tổng tiền
   - Click nút "Complete Purchase"
   - Hệ thống sẽ xử lý giao dịch và hiển thị thông báo thành công
4. **Xem lịch sử đơn hàng**:
   - Scroll xuống phần "Order History"
   - Click "Refresh Orders" để xem đơn hàng vừa tạo
   - Kiểm tra số lượng tồn kho đã được cập nhật

### Bước 4: Test các trường hợp

- ✅ Mua nhiều sản phẩm cùng lúc
- ✅ Mua số lượng lớn (kiểm tra validation stock)
- ✅ Xem lịch sử đơn hàng
- ✅ Kiểm tra stock được cập nhật sau khi mua

## Cách 2: Test qua API (Command Line)

### Sử dụng script test tự động

```bash
# Đảm bảo ứng dụng đang chạy
cd apps/postgres-demo
go run main.go

# Trong terminal khác, chạy script test
./test_purchase.sh
```

### Test thủ công với curl

#### 1. Login và lấy token

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

Lưu token từ response (ví dụ: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`)

#### 2. Xem danh sách sản phẩm

```bash
curl -X GET http://localhost:8080/api/products \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

#### 3. Mua hàng

```bash
curl -X POST http://localhost:8080/api/purchase \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "admin",
    "items": [
      {"product_id": 1, "quantity": 1},
      {"product_id": 2, "quantity": 2}
    ]
  }'
```

#### 4. Xem đơn hàng

```bash
curl -X GET "http://localhost:8080/api/orders?user_id=admin" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## Kiểm tra Database

### Xem dữ liệu trong PostgreSQL

```sql
-- Xem sản phẩm
SELECT * FROM products;

-- Xem đơn hàng
SELECT * FROM orders;

-- Xem chi tiết đơn hàng
SELECT oi.*, p.name as product_name 
FROM order_items oi 
JOIN products p ON oi.product_id = p.id 
ORDER BY oi.order_id;
```

## Test Cases

### ✅ Test Case 1: Mua hàng thành công
- Thêm sản phẩm vào giỏ
- Hoàn tất mua hàng
- **Kỳ vọng**: Đơn hàng được tạo, stock giảm, thông báo thành công

### ✅ Test Case 2: Mua hàng với stock không đủ
- Thêm sản phẩm với số lượng > stock
- **Kỳ vọng**: Lỗi "insufficient stock"

### ✅ Test Case 3: Mua nhiều sản phẩm
- Thêm nhiều sản phẩm khác nhau
- Hoàn tất mua hàng
- **Kỳ vọng**: Tất cả sản phẩm được xử lý trong 1 transaction

### ✅ Test Case 4: Xem lịch sử
- Sau khi mua hàng
- Xem Order History
- **Kỳ vọng**: Đơn hàng mới xuất hiện với đầy đủ thông tin

## Troubleshooting

### Lỗi: "database not initialized"
- Kiểm tra PostgreSQL đang chạy
- Kiểm tra connection string trong `application.properties`
- Đảm bảo database `fluxor_db` tồn tại

### Lỗi: "unauthorized" hoặc 401
- Kiểm tra token còn hợp lệ (24 giờ)
- Đăng nhập lại để lấy token mới

### Lỗi: "insufficient stock"
- Stock đã hết, cần reset database hoặc seed lại products
- Hoặc mua với số lượng nhỏ hơn

### Products không hiển thị
- Kiểm tra bảng `products` có dữ liệu
- Chạy lại ứng dụng để seed products

## Reset Database (nếu cần)

```sql
-- Xóa tất cả dữ liệu
TRUNCATE TABLE order_items CASCADE;
TRUNCATE TABLE orders CASCADE;
TRUNCATE TABLE products CASCADE;

-- Seed lại products (sẽ tự động khi khởi động app)
```
