// Package hoadondientu provides a connector for Vietnam e-invoice (Hóa đơn điện tử)
// systems. It supports creating, publishing, listing, and downloading e-invoices
// in compliance with Vietnamese tax regulations (e.g. Circular 78/2021/TT-BTC).
//
// # Providers
//
// The connector is designed for MISA meInvoice (https://doc.meinvoice.vn) by default.
// You can use other providers (VNPT eInvoice, eHoaDon Online, etc.) by setting
// BaseURL and optionally Provider to "custom".
//
// # Configuration
//
//	config := hoadondientu.DefaultConfig()
//	config.Token = "your-bearer-token"
//	// Or use username/password to obtain token:
//	config.Username = "integration_user"
//	config.Password = "integration_password"
//	config.BaseURL = "https://testapi.meinvoice.vn"  // test; prod: https://api.meinvoice.vn
//
//	comp := hoadondientu.NewComponent(config)
//	if err := comp.Start(fluxorCtx); err != nil { ... }
//	invoices, _ := comp.Invoices()
//	inv, _ := invoices.Create(ctx, &hoadondientu.CreateInvoiceParams{ ... })
//
// # Environment variables
//
//	HOADONDIENTU_BASE_URL   - API base URL (default: MISA test)
//	HOADONDIENTU_TOKEN      - Bearer token (optional if username/password set)
//	HOADONDIENTU_USERNAME   - Username for token login
//	HOADONDIENTU_PASSWORD   - Password for token login
//	HOADONDIENTU_PROVIDER   - "misa" | "custom"
//	HOADONDIENTU_TIMEOUT    - Request timeout (e.g. 30s)
package hoadondientu
