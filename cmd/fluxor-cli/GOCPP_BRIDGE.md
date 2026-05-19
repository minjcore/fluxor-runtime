# Go Calling C++ Integration Guide (C++20)

This directory demonstrates how to call **C++20** code from Go using CGO (C bindings).

## Overview

The C++ bridge lives in **base/private/cpp_bridge/** (low-level base). The CLI uses it via import.

- **base/private/cpp_bridge/cpp_wrapper.cpp**: C++20 implementation with functions wrapped in `extern "C"`
- **base/private/cpp_bridge/cpp_wrapper.h**: C header file declaring the C-compatible functions
- **base/private/cpp_bridge/cpp_bridge.go**: Go package using CGO to call C++ functions
- **cmd/fluxor-cli/cpp_example.go**: Example usage (imports `github.com/fluxorio/fluxor/base/private/cpp_bridge`)

This implementation focuses on **C++20** standard features including:
- `std::string_view` for efficient string handling
- `std::optional` for optional values
- Modern STL algorithms
- Improved type inference and structured bindings support

## Architecture

```
Go (cmd/fluxor-cli) → base/private/cpp_bridge (Go + CGO)
    ↓
CGO (C bindings)
    ↓
C Interface (cpp_wrapper.h)
    ↓
C++ Implementation (cpp_wrapper.cpp)
```

## Building

### Option 1: Using Makefile (from cmd/fluxor-cli)

```bash
cd cmd/fluxor-cli
make
make example
make clean
make install
```

### Option 2: Using Go directly (from repo root)

```bash
# Build with CGO (compiles base/private/cpp_bridge)
go build -o fluxor-cli ./cmd/fluxor-cli

# Run the C++ example
./fluxor-cli cpp-example
```

### Option 3: Using go install

```bash
go install .
fluxor-cli cpp-example
```

## Prerequisites

- **Go** (1.13+) with CGO enabled
- **C++ compiler** (g++ or clang++) with **C++20** support
- **C++20** standard library support

On macOS:
```bash
# g++ is usually available via Xcode Command Line Tools
xcode-select --install
```

On Linux:
```bash
sudo apt-get install build-essential g++  # Debian/Ubuntu
sudo yum install gcc-c++                  # RHEL/CentOS
```

On Windows:
- Install MinGW-w64 or use Visual Studio with C++ support

## Features Demonstrated

### 1. Simple Arithmetic Operations

```go
sum := CppAdd(42, 8)              // Returns 50
product := CppMultiply(3.14, 2.71) // Returns 8.51
```

### 2. String Operations (C++20 string_view)

```go
result := CppConcatenate("Hello, ", "C++20 from Go!")
// Returns "Hello, C++20 from Go!"

// C++20: Using string_view for efficient processing
processed := CppProcessStringView("hello c++20")
// Returns "PROCESSED: HELLO C++20"
```

### 3. Array Operations

```go
numbers := []int{1, 2, 3, 4, 5}
sum := CppSumArray(numbers) // Returns 15

// C++20: Finding maximum using modern algorithms
maxVal := CppFindMax([]int{45, 12, 78, 23, 91}) // Returns 91
```

### 4. C++20 std::optional

```go
// Demonstrates C++20 optional values
opt1 := CppOptionalValue(42.5, true)  // Returns "Value: 42.500000"
opt2 := CppOptionalValue(0.0, false)  // Returns "No value"
```

### 5. C++ Class Wrapper (C++20 features)

```go
calc := NewCalculator()
defer calc.Close()  // Always close to prevent memory leaks

calc.SetValue(10.0)
calc.Calculate("add", 5.0)      // Result: 15.0 (uses string_view internally)
calc.Calculate("multiply", 3.0) // Result: 45.0
calc.Calculate("power", 2.0)    // Result: 2025.0
```

## CGO Configuration (C++20)

The CGO directives in `cpp_bridge.go` configure the build for **C++20**:

```go
/*
#cgo CXXFLAGS: -std=c++20
#cgo LDFLAGS: -lstdc++

#include "cpp_wrapper.h"
#include <stdlib.h>
*/
```

- `CXXFLAGS`: C++ compiler flags (**C++20** standard)
- `LDFLAGS`: Linker flags (link C++ standard library)
- `#include`: Header files needed

## C++20 Features Used

### 1. std::string_view
Efficient non-owning string reference that avoids unnecessary allocations:

```cpp
std::string_view sv(input);
std::string result(sv);  // No copy until needed
```

### 2. std::optional
Represents optional values without using pointers or special sentinel values:

```cpp
std::optional<double> opt = (value > 0) ? std::optional<double>(value) : std::nullopt;
if (opt.has_value()) {
    // Use opt.value()
}
```

### 3. Modern STL Algorithms
Using C++20 improved algorithm support:

```cpp
int* maxElement = std::max_element(arr, arr + length);
```

### 4. Improved String Handling
The Calculator class uses `string_view` for efficient operation comparisons.

## Memory Management

### Automatic Memory Management

Go's `defer` and CGO handle most memory automatically:

```go
// C.CString allocates memory, C.free releases it
cstr := C.CString("hello")
defer C.free(unsafe.Pointer(cstr))
```

### Manual Memory Management

For C++ objects, always call `Close()`:

```go
calc := NewCalculator()
defer calc.Close()  // Releases C++ object
```

### String Memory

The `CppConcatenate` function automatically manages C++ string memory:

```go
// Memory is automatically freed
result := CppConcatenate("str1", "str2")
```

## Extending the Example

### Adding New C++20 Functions

1. **Add function to C++ file** (`cpp_wrapper.cpp`) using C++20 features:
```cpp
#include <optional>
#include <string_view>

extern "C" {
    int MyNewFunction(int arg) {
        // Your C++20 code here
        std::optional<int> opt = arg > 0 ? std::optional<int>(arg) : std::nullopt;
        return opt.value_or(0) * 2;
    }
}
```

2. **Add declaration to header** (`cpp_wrapper.h`):
```c
int MyNewFunction(int arg);
```

3. **Create Go wrapper** (`cpp_bridge.go`):
```go
func CppMyNewFunction(arg int) int {
    return int(C.MyNewFunction(C.int(arg)))
}
```

### Adding New C++ Classes

1. **Create C++ class** (`cpp_wrapper.cpp`):
```cpp
class MyClass {
public:
    void doSomething() { /* ... */ }
};

extern "C" {
    void* CreateMyClass() { return new MyClass(); }
    void DestroyMyClass(void* obj) { delete static_cast<MyClass*>(obj); }
    void DoSomething(void* obj) { static_cast<MyClass*>(obj)->doSomething(); }
}
```

2. **Add Go wrapper** (`cpp_bridge.go`):
```go
type MyClass struct {
    ptr unsafe.Pointer
}

func NewMyClass() *MyClass {
    return &MyClass{ptr: C.CreateMyClass()}
}

func (m *MyClass) DoSomething() {
    C.DoSomething(m.ptr)
}

func (m *MyClass) Close() {
    if m.ptr != nil {
        C.DestroyMyClass(m.ptr)
        m.ptr = nil
    }
}
```

## Troubleshooting

### Build Errors

**Error: "cannot find -lstdc++"**
- Ensure C++ standard library is installed
- On macOS, install Xcode Command Line Tools
- On Linux, install `g++` package

**Error: "cpp_wrapper.h: No such file or directory"**
- Ensure header file is in the same directory as Go files
- Check `#include` paths in CGO comments

**Error: "undefined reference"**
- Ensure all C++ functions are declared in header file
- Verify `extern "C"` wrapper is correct

### Runtime Errors

**Segmentation fault**
- Check for memory leaks (always call `Close()`)
- Verify pointer is not nil before use
- Ensure C++ objects are not accessed after destruction

**Incorrect results**
- Verify data type conversions (int vs int32, float64 vs double)
- Check endianness for complex data structures

## Best Practices

1. **Always use `defer` for cleanup**:
   ```go
   calc := NewCalculator()
   defer calc.Close()
   ```

2. **Validate inputs** before passing to C++:
   ```go
   if arr == nil || len(arr) == 0 {
       return 0
   }
   ```

3. **Use type conversions explicitly**:
   ```go
   C.int(value)  // Not just value
   ```

4. **Keep C++ interface simple**: Complex types are harder to bridge

5. **Document memory ownership**: Who allocates and who frees

## Testing

Run the example:
```bash
./fluxor-cli cpp-example
```

Expected output:
```
=== Go Calling C++ Example ===

1. Simple Arithmetic:
   42 + 8 = 50
   3.14 * 2.71 = 8.51

2. String Concatenation:
   "Hello, " + "C++ from Go!" = "Hello, C++ from Go!"

3. Array Sum:
   Sum of [1 2 3 4 5 6 7 8 9 10] = 55

4. C++ Calculator Class:
   Initial value: 10.00
   After add(5): 15.00
   After multiply(3): 45.00
   After subtract(2): 43.00
   After power(2): 1849.00
   After divide(4): 462.25

5. C++20 Features:
   a) String View Processing:
      Input: "hello c++20"
      Output: PROCESSED: HELLO C++20

   b) Finding Maximum (C++20 algorithms):
      Array: [45 12 78 23 91 56 34 67]
      Maximum: 91

   c) Optional Values (C++20 std::optional):
      Optional(42.5, true): Value: 42.500000
      Optional(0.0, false): No value
      Optional(-5.0, true): No value

=== Example Complete ===
```

## References

- [CGO Documentation](https://golang.org/cmd/cgo/)
- [Go CGO Examples](https://github.com/golang/go/wiki/cgo)
- [CGO Best Practices](https://dave.cheney.net/2016/01/18/cgo-is-not-go)
