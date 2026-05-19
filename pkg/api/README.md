# API – Controller / Service / Entity / Logic (Interface)

API được tổ chức theo module: **api/a**, **api/b**, **api/c**. Mỗi module chứa **entity**, **logic**, **service**, **controller** và **đóng gói thành interface** để dễ inject và test.

## Interfaces (mỗi module)

| Interface          | Mục đích                         | Implemented by |
|--------------------|----------------------------------|----------------|
| **LogicDoer**      | Business/use-case: `Do(*Entity)(*Entity, error)` | `*Logic` |
| **ServiceProcessor** | Orchestration: `Process(*Entity)(*Entity, error)` | `*Service` |
| **ControllerHandler** | Input layer: `Service() ServiceProcessor` | `*Controller` |

- **Service** phụ thuộc vào **LogicDoer** (không phụ thuộc `*Logic` cụ thể).
- **Controller** phụ thuộc vào **ServiceProcessor** (không phụ thuộc `*Service` cụ thể).
- Có thể inject mock khi test: `NewServiceWithLogic(mockLogic)`, `NewControllerWithService(mockSvc)`.

## Cấu trúc

```
pkg/api/
├── a/
│   ├── interfaces.go  # LogicDoer, ServiceProcessor, ControllerHandler
│   ├── entity.go      # Entity
│   ├── logic.go       # Logic (implements LogicDoer)
│   ├── service.go     # Service (implements ServiceProcessor, depends on LogicDoer)
│   └── controller.go  # Controller (implements ControllerHandler, depends on ServiceProcessor)
├── b/
│   └── ...
├── c/
│   └── ...
└── README.md
```

## Luồng

```
Request → Controller (ControllerHandler) → Service (ServiceProcessor) → Logic (LogicDoer)
                ↑                                    ↓
              Entity ←───────────────────────────────┘
```

## Dùng / inject

```go
// Mặc định (concrete)
ctrl := a.NewController()

// Inject service (test / DI)
mockSvc := &mockServiceProcessor{...}
ctrl := a.NewControllerWithService(mockSvc)

// Inject logic vào service (test / DI)
mockLogic := &mockLogicDoer{...}
svc := a.NewServiceWithLogic(mockLogic)
ctrl := a.NewControllerWithService(svc)
```

## Thêm module mới

Tạo thư mục `pkg/api/<tên>/` với: `interfaces.go`, `entity.go`, `logic.go`, `service.go`, `controller.go`, theo pattern của `a`, `b`, `c`.
