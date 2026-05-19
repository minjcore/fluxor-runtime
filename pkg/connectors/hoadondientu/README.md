# Hóa đơn điện tử (Vietnam E-Invoice) Connector

Connector tích hợp **Hóa đơn điện tử** Việt Nam với Fluxor, tương thích **MISA meInvoice** và có thể cấu hình cho các nhà cung cấp khác (VNPT, eHoaDon, ...).

## Tài liệu tham khảo

- [MISA meInvoice – Tài liệu tích hợp](https://doc.meinvoice.vn/)
- [Đặc tả API Service MISA meInvoice](https://doc.meinvoice.vn/webapi/)
- [eHoaDon Online – Kết nối API](https://docs.ehoadon.online/cac-buoc-ket-noi-api)
- Nghị định / Thông tư: Circular 78/2021/TT-BTC (định dạng XML, chữ ký số)

## Cấu hình

| Biến môi trường | Mô tả | Mặc định |
|-----------------|--------|----------|
| `HOADONDIENTU_BASE_URL` | Base URL API (MISA test/prod hoặc provider khác) | `https://testapi.meinvoice.vn` |
| `HOADONDIENTU_TOKEN` | Bearer token (nếu dùng token trực tiếp) | - |
| `HOADONDIENTU_USERNAME` | Tài khoản tích hợp (để lấy token) | - |
| `HOADONDIENTU_PASSWORD` | Mật khẩu tích hợp | - |
| `HOADONDIENTU_PROVIDER` | `misa` \| `custom` | `misa` |
| `HOADONDIENTU_TIMEOUT` | Timeout gọi API | `30s` |

**Lưu ý:** Cần ít nhất một trong hai: `HOADONDIENTU_TOKEN` hoặc cặp `HOADONDIENTU_USERNAME` / `HOADONDIENTU_PASSWORD`.

## Sử dụng nhanh

```go
package main

import (
    "context"
    "log"

    "github.com/fluxorio/fluxor/pkg/connectors/hoadondientu"
    "github.com/fluxorio/fluxor/pkg/core"
)

func main() {
    cfg := hoadondientu.DefaultConfig()
    cfg.Username = "your_username"
    cfg.Password = "your_password"
    cfg.BaseURL = "https://testapi.meinvoice.vn"

    comp := hoadondientu.NewComponent(cfg)
    ctx := core.NewFluxorContext(context.Background(), nil, nil)
    if err := comp.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer comp.Stop(ctx)

    invClient, _ := comp.Invoices()

    // Lấy danh sách mẫu hóa đơn
    templates, err := invClient.ListTemplates(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Tạo hóa đơn (bản nháp)
    inv, err := invClient.Create(context.Background(), &hoadondientu.CreateInvoiceParams{
        TemplateCode:  "01GTKT",
        Symbol:        "AA/24",
        IssueDate:     "2024-01-15",
        KindOfService: "1",
        Currency:      "VND",
        Buyer: &hoadondientu.Buyer{
            TaxCode: "0123456789",
            Name:    "Công ty ABC",
            Address: "123 Đường XYZ",
            Email:   "contact@abc.vn",
        },
        Items: []hoadondientu.InvoiceItem{
            {
                Name:      "Dịch vụ tư vấn",
                Quantity:  1,
                Unit:      "lần",
                UnitPrice: 10000000,
                Amount:    10000000,
                VATRate:   10,
                VATAmount: 1000000,
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Phát hành lên CQT
    published, err := invClient.Publish(context.Background(), inv.ID)
    if err != nil {
        log.Fatal(err)
    }

    // Tải PDF
    pdf, _ := invClient.DownloadPDF(context.Background(), published.ID)

    // Gửi email cho khách
    _ = invClient.SendEmail(context.Background(), published.ID, "contact@abc.vn")
}
```

## API chính

- **Invoices:** `Create`, `Get`, `List`, `Update`, `Delete`, `GetStatus`, `Publish`
- **Download:** `DownloadPDF`, `DownloadXML`
- **Send:** `SendEmail`
- **Templates:** `ListTemplates` (mẫu số / ký hiệu)

## Đăng ký connector

```go
import "github.com/fluxorio/fluxor/pkg/connectors"
import "github.com/fluxorio/fluxor/pkg/connectors/hoadondientu"

comp := hoadondientu.NewComponent(hoadondientu.DefaultConfig())
_ = connectors.Register(comp)
```

## Provider khác (VNPT, eHoaDon, ...)

Đặt `Provider: hoadondientu.ProviderCustom` và `BaseURL` trỏ tới endpoint của nhà cung cấp. Đường dẫn API có thể khác (ví dụ prefix path); nếu cần có thể mở rộng config thêm `APIPrefix` hoặc mapping endpoint trong client.
