# Tổng hợp lệnh fluxor-cli

Chạy từ repo root: `./fluxor-cli` hoặc `fluxor-cli` (nếu đã `go install`).

---

## 1. Local — PM2-like (process trên máy mình)

| Lệnh | Mô tả |
|------|--------|
| `fluxor-cli new <appname>` | Tạo app mới (main.go, config.json, go.mod) |
| `fluxor-cli new service <name>` | Tạo service mới (Eureka) |
| `fluxor-cli new workflow <name>` | Tạo file workflow JSON |
| `fluxor-cli start <name> [dir]` | Chạy app nền (daemon), log → ~/.fluxor-cli/<name>.log |
| `fluxor-cli stop <name>` | Dừng app |
| `fluxor-cli restart <name>` | Restart app (local) |
| `fluxor-cli list` / `fluxor-cli ls` | Liệt kê app đang quản lý |
| `fluxor-cli logs <name> [--lines N]` | Xem log app |
| `fluxor-cli status <name>` | Trạng thái app (PM2-like) |
| `fluxor-cli delete <name>` | Xóa app khỏi danh sách (không xóa process đang chạy) |

---

## 2. Deploy / VPS (target trong deploy.yaml)

| Lệnh | Mô tả |
|------|--------|
| `fluxor-cli deploy -target <target> -go-app -force` | Deploy binary Go + systemd + config.json |
| `fluxor-cli deploy -target <target> -node-app -force` | Deploy build Node (HTML/assets) |
| `fluxor-cli deploy -target <target> -nginx -force` | Deploy config Nginx (repo/*.conf) |
| `fluxor-cli deploy -target <target> -certbot -force` | Chạy certbot trên VPS (SSL Let's Encrypt) |
| `fluxor-cli deploy -target <target> -docker-compose -force` | Deploy Docker Compose |
| `fluxor-cli deploy -target <target> -go-app -nginx -force` | Deploy cả Go app + Nginx |

**Flags chung:** `-config <path>`, `-env-file <path>`, `-force` (bỏ confirm).

---

## 3. Restart / Undeploy trên VPS

| Lệnh | Mô tả |
|------|--------|
| `fluxor-cli restart -target <target> -go-app -force` | Restart systemd service Go app |
| `fluxor-cli restart -target <target> -nginx -force` | Restart nginx |
| `fluxor-cli undeploy -target <target> -go-app [-remove-files] -force` | Gỡ Go app (và tùy chọn xóa file) |
| `fluxor-cli undeploy -target <target> -nginx -force` | Gỡ config Nginx |

---

## 4. Xem trạng thái / log trên VPS

| Lệnh | Mô tả |
|------|--------|
| `fluxor-cli state -target <target>` | Trạng thái tổng (services, resources, health) |
| `fluxor-cli state -target <target> -json` | Trạng thái dạng JSON |
| `fluxor-cli list-services -target <target>` | Liệt kê systemd services (pipe to grep) |
| `fluxor-cli list-services -target <target> \| grep nginx` | Lọc service tên chứa "nginx" |
| `fluxor-cli service-logs -target <target> [-service <name>] [-lines N]` | journalctl cho service (mặc định: go_app.service_name) |

**Ví dụ quadgate-io:**

```bash
fluxor-cli state -target quadgate-io
fluxor-cli list-services -target quadgate-io | grep fluxor
fluxor-cli service-logs -target quadgate-io -lines 50
fluxor-cli service-logs -target quadgate-io -service fluxor-quadgate_io -lines 100
```

---

## 5. Khác

| Lệnh | Mô tả |
|------|--------|
| `fluxor-cli up staticsite [-dir=.] [-port=8080]` | Chạy static site local (thư mục hiện tại hoặc -dir) |
| `fluxor-cli version` | In phiên bản |
| `fluxor-cli cpp-example` | Chạy ví dụ Go gọi C++ (cần build C++ bridge) |

---

## 6. Chuẩn bị trước khi dùng deploy/restart/state

- **deploy.yaml** ở repo root (hoặc -config chỉ đường dẫn).
- **.env.local** (repo root) nếu dùng sshpass:
  - `SSH_HOST`, `SSH_USER`, `SSH_PASSWORD`
  - `CERTBOT_EMAIL` (cho deploy -certbot)
- Hoặc dùng SSH key: `ssh_key` trong deploy.yaml + `ssh-copy-id root@<host>`.

---

## 7. Target quadgate-io (tóm tắt)

```bash
# Deploy
./fluxor-cli deploy -target quadgate-io -go-app -force    # binary + config
./fluxor-cli deploy -target quadgate-io -nginx -force    # repo/quadgate.io.conf
./fluxor-cli deploy -target quadgate-io -certbot -force  # SSL (cần CERTBOT_EMAIL)

# Trạng thái & log
./fluxor-cli state -target quadgate-io
./fluxor-cli service-logs -target quadgate-io -lines 100

# Restart
./fluxor-cli restart -target quadgate-io -go-app -force
./fluxor-cli restart -target quadgate-io -nginx -force
```

Config Nginx: `repo/quadgate.io.conf` (version trong comment đầu file).
