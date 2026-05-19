# Admin UI (P1 – Shell)

**Website đang build** = Admin UI này: giao diện quản lý chung (Dashboard, Storage, CI, sau: PaaS/Cluster, …).

Font: **Urbanist** (Google Fonts), cùng font với fluxor-font.

UI quản lý chung: layout (sidebar, header) và các trang placeholder cho Dashboard, Storage, CI.

## Routes

- `GET /admin` → redirect `302` to `/admin/dashboard`
- `GET /admin/dashboard` → Admin shell, nội dung Dashboard (placeholder; P2 embed dashboard)
- `GET /admin/storage` → Admin shell, nội dung Storage (placeholder; P4)
- `GET /admin/ci` → Admin shell, nội dung CI (placeholder; P5)
- `GET /admin/admin.js`, `GET /admin/admin.css` → static assets

Nếu dashboard verticle dùng `Prefix` (ví dụ `/app`), admin nằm tại `/app/admin`, `/app/admin/dashboard`, …

## Tích hợp

Admin được đăng ký tự động khi deploy **DashboardVerticle** (cùng router, cùng prefix). Không cần verticle riêng.

## Kế hoạch

- **P1** (done): Shell, sidebar, placeholder.
- **P2**: Embed dashboard hiện tại vào `/admin/dashboard`.
- **P3**: API Storage.
- **P4**: Trang Storage (list keys, get/put/delete).
- **P5**: Trang CI (workflows, run, executions).

Xem `docs/ADMIN_UI_PLAN.md`.
