# MVC Architecture - PostgreSQL Demo

Dự án đã được tách thành kiến trúc MVC (Model-View-Controller) để dễ bảo trì và mở rộng.

## 📁 Cấu trúc thư mục

```
apps/postgres-demo/
├── models/              # Data models (domain entities)
│   ├── auth.go         # LoginRequest, LoginResponse
│   ├── order.go        # Order, OrderItem
│   ├── product.go      # Product
│   └── purchase.go     # PurchaseRequest, PurchaseItem, PurchaseResponse
│
├── views/              # HTML templates (presentation layer)
│   ├── login.go        # Login page HTML
│   └── dashboard.go   # Dashboard page HTML
│
├── controllers/        # Request handlers (HTTP layer)
│   ├── auth_controller.go      # Authentication endpoints
│   ├── purchase_controller.go  # Purchase endpoints
│   ├── dashboard_controller.go # Dashboard page
│   └── page_controller.go      # Public pages
│
├── services/           # Business logic layer
│   ├── auth_service.go        # Authentication logic
│   └── purchase_service.go   # Purchase transaction logic
│
├── routes.go          # Route definitions (MVC wiring)
├── main.go            # Application entry point
└── persistence_test.go # Tests
```

## 🏗️ Kiến trúc MVC

### Models (`models/`)
**Trách nhiệm**: Định nghĩa cấu trúc dữ liệu

- `Product`: Thông tin sản phẩm
- `Order`: Đơn hàng
- `OrderItem`: Chi tiết đơn hàng
- `PurchaseRequest`: Request mua hàng
- `PurchaseResponse`: Response sau khi mua
- `LoginRequest`: Request đăng nhập
- `LoginResponse`: Response đăng nhập

### Views (`views/`)
**Trách nhiệm**: HTML templates và UI

- `GetLoginHTML()`: Trang đăng nhập
- `GetDashboardHTML()`: Trang dashboard

### Controllers (`controllers/`)
**Trách nhiệm**: Xử lý HTTP requests, gọi services, trả về responses

- `AuthController`: Xử lý login/logout
- `PurchaseController`: Xử lý mua hàng, sản phẩm, đơn hàng
- `DashboardController`: Hiển thị dashboard
- `PageController`: Hiển thị các trang public

### Services (`services/`)
**Trách nhiệm**: Business logic, database operations

- `AuthService`: Xác thực người dùng, tạo JWT token
- `PurchaseService`: Xử lý giao dịch mua hàng với ACID guarantees

### Routes (`routes.go`)
**Trách nhiệm**: Định nghĩa routes, kết nối controllers với middleware

- Đăng ký routes
- Áp dụng JWT middleware
- Kết nối controllers với services

## 🔄 Luồng xử lý request

```
HTTP Request
    ↓
Routes (routes.go)
    ↓
Controller (controllers/)
    ↓
Service (services/)
    ↓
Model (models/)
    ↓
Database
```

### Ví dụ: Purchase Flow

1. **Request**: `POST /api/purchase`
2. **Route**: `routes.go` → `purchaseController.MakePurchase`
3. **Controller**: `PurchaseController.MakePurchase()`
   - Validate request
   - Extract user from JWT
   - Call service
4. **Service**: `PurchaseService.MakePurchase()`
   - Begin transaction
   - Validate products
   - Create order
   - Update stock
   - Commit transaction
5. **Response**: Controller trả về JSON

## 📝 Code Examples

### Controller Example

```go
// controllers/purchase_controller.go
func (c *PurchaseController) MakePurchase(ctx *web.FastRequestContext) error {
    var req models.PurchaseRequest
    if err := ctx.BindJSON(&req); err != nil {
        return ctx.JSON(400, map[string]interface{}{
            "error": "invalid_request",
        })
    }
    
    result, err := c.purchaseService.MakePurchase(ctx.Context(), req)
    if err != nil {
        return ctx.JSON(400, map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    return ctx.JSON(200, result)
}
```

### Service Example

```go
// services/purchase_service.go
func (s *PurchaseService) MakePurchase(ctx context.Context, req models.PurchaseRequest) (*models.PurchaseResponse, error) {
    // Business logic here
    // Transaction management
    // Database operations
    return response, nil
}
```

## ✅ Lợi ích của MVC

1. **Separation of Concerns**: Mỗi layer có trách nhiệm rõ ràng
2. **Testability**: Dễ test từng component riêng biệt
3. **Maintainability**: Dễ bảo trì và mở rộng
4. **Reusability**: Services có thể được sử dụng lại
5. **Scalability**: Dễ thêm features mới

## 🚀 Thêm tính năng mới

### Bước 1: Tạo Model
```go
// models/user.go
type User struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}
```

### Bước 2: Tạo Service
```go
// services/user_service.go
type UserService struct {
    db *dbruntime.DatabaseComponent
}

func (s *UserService) GetUser(id int) (*models.User, error) {
    // Business logic
}
```

### Bước 3: Tạo Controller
```go
// controllers/user_controller.go
type UserController struct {
    userService *services.UserService
}

func (c *UserController) GetUser(ctx *web.FastRequestContext) error {
    // Handle request
}
```

### Bước 4: Đăng ký Route
```go
// routes.go
userController := controllers.NewUserController(userService)
router.GETFastWith("/api/users/:id", userController.GetUser, jwtMiddleware)
```

## 📚 Best Practices

1. **Models**: Chỉ chứa data structures, không có logic
2. **Controllers**: Chỉ xử lý HTTP, không có business logic
3. **Services**: Chứa tất cả business logic
4. **Views**: Chỉ chứa HTML/CSS/JS, không có logic
5. **Routes**: Chỉ định nghĩa routes và middleware

## 🔍 File Mapping

| Chức năng | File cũ | File mới |
|-----------|---------|----------|
| Models | `purchase.go` | `models/*.go` |
| Services | `purchase.go` | `services/*.go` |
| Controllers | `webui.go` | `controllers/*.go` |
| Views | `webui.go` | `views/*.go` |
| Routes | `webui.go` | `routes.go` |
