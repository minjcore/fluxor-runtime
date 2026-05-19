# Tại Sao OS Threads Không Giảm Xuống?

## 📊 Tình Trạng Hiện Tại

Từ metrics dashboard, bạn quan sát thấy:
- **GOMAXPROCS:** 8 (số OS threads tối đa cho Go code)
- **OS Threads:** 17 (số OS threads thực tế)
- **Goroutines:** 1,131

**Vấn đề:** OS Threads (17) > GOMAXPROCS (8) và không giảm xuống khi load giảm.

## ✅ Đây Là Behavior Bình Thường Của Go Runtime

### 1. **Go Runtime Giữ Lại OS Threads (Thread Pooling)**

Go runtime **không destroy OS threads ngay lập tức** khi load giảm. Thay vào đó, nó giữ lại threads trong một pool để reuse. Đây là **optimization** để:

- ✅ **Tránh overhead:** Tạo/destroy threads tốn kém (syscall overhead)
- ✅ **Cải thiện performance:** Reuse threads nhanh hơn tạo mới
- ✅ **Giảm latency:** Khi có blocking I/O, threads sẵn sàng xử lý ngay

**Threads sẽ giảm chậm dần** theo thời gian (thường sau vài phút idle), không phải ngay lập tức.

### 2. **OS Threads > GOMAXPROCS Là Bình Thường**

**17 threads với GOMAXPROCS=8 là hợp lý** vì:

```
OS Threads (17) = GOMAXPROCS (8) + Blocking I/O Threads (9)
```

**Các threads bổ sung đến từ:**

1. **Blocking I/O (Network/File):**
   - HTTP server: Khi goroutine block trên network I/O (read/write)
   - File I/O: Khi goroutine block trên file operations
   - DNS lookups: Khi goroutine block trên DNS queries

2. **CGO Calls:**
   - Nếu có CGO code, mỗi CGO call có thể tạo thêm thread

3. **Runtime Threads:**
   - GC threads (garbage collector)
   - Finalizer threads
   - Signal handler threads
   - Scheduler threads

### 3. **Khi Nào OS Threads Sẽ Giảm?**

Go runtime sẽ **từ từ giảm số threads** khi:

- ✅ **Idle time dài:** Threads idle vài phút không được dùng
- ✅ **No blocking I/O:** Không có blocking syscalls
- ✅ **Low goroutine count:** Số goroutines giảm đáng kể

**Timeline điển hình:**
- **Ngay lập tức:** Threads không giảm (giữ để reuse)
- **Sau 1-2 phút idle:** Threads có thể giảm một vài
- **Sau 5-10 phút idle:** Threads có thể giảm về gần GOMAXPROCS

### 4. **Memory Impact**

Mỗi OS thread tốn:
- **Stack size:** ~2MB per thread (default)
- **Total với 17 threads:** ~34MB stack memory

**17 threads với GOMAXPROCS=8:**
- **Extra threads:** 9 threads
- **Extra memory:** ~18MB (không đáng kể)
- **Performance impact:** Minimal (threads idle không tốn CPU)

## 🔍 Khi Nào Cần Lo Lắng?

### ✅ **Không Phải Vấn Đề (Bình Thường):**
- OS Threads = GOMAXPROCS * 1.5 - 2.5 (ví dụ: 8 * 2 = 16 threads)
- OS Threads tăng khi có load (blocking I/O)
- OS Threads giữ ở mức cao sau khi load giảm (thread pooling)

### ⚠️ **Có Thể Là Vấn Đề:**
- OS Threads > GOMAXPROCS * 5 (ví dụ: 8 * 5 = 40 threads)
- OS Threads tăng liên tục không dừng (potential thread leak)
- OS Threads > 100 (có thể có vấn đề với blocking code)

### 🚨 **Chắc Chắn Là Vấn Đề:**
- OS Threads tăng đến hàng nghìn
- OS Threads tăng liên tục không dừng
- Memory usage tăng đáng kể do threads

## 📈 Metrics Của Bạn

**Hiện tại:**
- GOMAXPROCS: 8
- OS Threads: 17
- Ratio: 17/8 = 2.125x (bình thường!)

**Đánh Giá:**
✅ **Hoàn toàn bình thường!**

17 threads với GOMAXPROCS=8 là ratio hợp lý (2.125x) khi có:
- HTTP server (blocking network I/O)
- File I/O (nếu có)
- Runtime threads (GC, finalizers)

## 🎯 Kết Luận

**Số OS Threads không giảm ngay lập tức là behavior bình thường của Go runtime.**

**Không cần lo lắng nếu:**
- OS Threads < GOMAXPROCS * 3 (ví dụ: 8 * 3 = 24)
- OS Threads không tăng liên tục
- Memory usage ổn định

**17 threads với GOMAXPROCS=8 là healthy và optimal cho HTTP server workload!**

---

## 📚 Tài Liệu Tham Khảo

- [Go Runtime: Thread Management](https://go.dev/src/runtime/proc.go)
- [Go Scheduler: GOMAXPROCS vs OS Threads](https://golang.org/doc/effective_go#goroutines)
- [Understanding Go Runtime](https://www.ardanlabs.com/blog/2018/08/scheduling-in-go-part1.html)
