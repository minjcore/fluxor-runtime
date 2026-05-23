// C++ wrapper implementation for CGO. Uses C++20.

#include "cpp_wrapper.h"
#include <algorithm>
#include <cmath>
#include <cstring>
#include <memory>
#include <numeric>
#include <optional>
#include <string>
#include <string_view>
#include <variant>
#include <iostream>
#include <vector>

extern "C" {

// Simple C-style functions
int Add(int a, int b) {
    return a + b;
}

double Multiply(double a, double b) {
    return a * b;
}

// String concatenation using C++20 string_view (caller must free the returned string)
const char* Concatenate(const char* str1, const char* str2) {
    std::string_view sv1(str1);
    std::string_view sv2(str2);
    std::string result(sv1);
    result += sv2;
    char* cstr = new char[result.length() + 1];
    std::strcpy(cstr, result.c_str());
    return cstr;
}

void FreeString(char* str) {
    delete[] str;
}

// C++20: Using string_view for efficient string processing
const char* ProcessStringView(const char* input) {
    std::string_view sv(input);
    std::string result = "Processed: ";
    result += sv;
    std::transform(result.begin(), result.end(), result.begin(), ::toupper);
    char* cstr = new char[result.length() + 1];
    std::strcpy(cstr, result.c_str());
    return cstr;
}

// C++20: Using std::max_element with modern C++ features
int FindMax(int* arr, int length) {
    if (length <= 0) return 0;
    int* maxElement = std::max_element(arr, arr + length);
    return *maxElement;
}

// C++20: Using std::optional for optional return values
const char* OptionalValue(double value, int returnValue) {
    std::optional<double> opt = (value > 0) ? std::optional<double>(value) : std::nullopt;
    if (returnValue && opt.has_value()) {
        std::string result = "Value: " + std::to_string(opt.value());
        char* cstr = new char[result.length() + 1];
        std::strcpy(cstr, result.c_str());
        return cstr;
    }
    std::string result = opt.has_value() ? "Has value" : "No value";
    char* cstr = new char[result.length() + 1];
    std::strcpy(cstr, result.c_str());
    return cstr;
}

// Array operations
int SumArray(int* arr, int length) {
    int sum = 0;
    for (int i = 0; i < length; i++) {
        sum += arr[i];
    }
    return sum;
}

double* CreateDoubleArray(int length) {
    return new double[length];
}

void FreeDoubleArray(double* arr) {
    delete[] arr;
}

} // extern "C"

// C++ Calculator class using C++20 features
class Calculator {
private:
    double value;

public:
    Calculator() : value(0.0) {}

    void setValue(double v) {
        value = v;
    }

    double getValue() const {
        return value;
    }

    double calculate(const std::string& operation, double operand) {
        std::string_view op(operation);
        if (op == "add") {
            value += operand;
        } else if (op == "subtract") {
            value -= operand;
        } else if (op == "multiply") {
            value *= operand;
        } else if (op == "divide") {
            if (operand != 0.0) {
                value /= operand;
            }
        } else if (op == "power") {
            value = std::pow(value, operand);
        }
        return value;
    }
};

extern "C" {

void* CreateCalculator() {
    return new Calculator();
}

void DestroyCalculator(void* calc) {
    delete static_cast<Calculator*>(calc);
}

void SetValue(void* calc, double value) {
    static_cast<Calculator*>(calc)->setValue(value);
}

double GetValue(void* calc) {
    return static_cast<Calculator*>(calc)->getValue();
}

double Calculate(void* calc, const char* operation, double operand) {
    return static_cast<Calculator*>(calc)->calculate(std::string(operation), operand);
}

// --- Network utilities ---

static const char kBase64Chars[] =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

const char* URLParse(const char* url) {
    std::string u(url);
    std::string host, port, path;
    port = "80";
    path = "/";

    size_t scheme_end = u.find("://");
    size_t start = (scheme_end != std::string::npos) ? scheme_end + 3 : 0;

    size_t path_start = u.find('/', start);
    if (path_start != std::string::npos) {
        path = u.substr(path_start);
        u = u.substr(0, path_start);
    }

    size_t colon = u.find(':', start);
    if (colon != std::string::npos) {
        host = u.substr(start, colon - start);
        port = u.substr(colon + 1);
    } else {
        host = u.substr(start);
    }

    std::string result = host + "|" + port + "|" + path;
    char* out = new char[result.length() + 1];
    std::strcpy(out, result.c_str());
    return out;
}

const char* Base64Encode(const unsigned char* data, int length) {
    if (!data || length <= 0) {
        char* empty = new char[1];
        empty[0] = '\0';
        return empty;
    }
    std::string result;
    result.reserve(((length + 2) / 3) * 4);
    for (int i = 0; i < length; i += 3) {
        unsigned int n = static_cast<unsigned int>(data[i]) << 16;
        if (i + 1 < length) n |= static_cast<unsigned int>(data[i + 1]) << 8;
        if (i + 2 < length) n |= static_cast<unsigned int>(data[i + 2]);
        result += kBase64Chars[(n >> 18) & 63];
        result += kBase64Chars[(n >> 12) & 63];
        result += (i + 1 < length) ? kBase64Chars[(n >> 6) & 63] : '=';
        result += (i + 2 < length) ? kBase64Chars[n & 63] : '=';
    }
    char* out = new char[result.length() + 1];
    std::strcpy(out, result.c_str());
    return out;
}

static int Base64DecodeChar(char c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '+') return 62;
    if (c == '/') return 63;
    return -1;
}

unsigned char* Base64Decode(const char* b64, int* outLength) {
    *outLength = 0;
    if (!b64) return nullptr;
    size_t len = std::strlen(b64);
    if (len == 0) return nullptr;
    size_t out_len = (len / 4) * 3;
    if (b64[len - 1] == '=') out_len--;
    if (len > 1 && b64[len - 2] == '=') out_len--;
    unsigned char* out = new unsigned char[out_len + 1];
    out[out_len] = '\0';
    int j = 0;
    for (size_t i = 0; i + 4 <= len && j < static_cast<int>(out_len); i += 4) {
        int a = Base64DecodeChar(b64[i]);
        int b = Base64DecodeChar(b64[i + 1]);
        int c = Base64DecodeChar(b64[i + 2]);
        int d = Base64DecodeChar(b64[i + 3]);
        if (a < 0 || b < 0) break;
        out[j++] = static_cast<unsigned char>((a << 2) | (b >> 4));
        if (c >= 0 && j < static_cast<int>(out_len))
            out[j++] = static_cast<unsigned char>((b << 4) | (c >> 2));
        if (d >= 0 && j < static_cast<int>(out_len))
            out[j++] = static_cast<unsigned char>((c << 6) | d);
    }
    *outLength = j;
    return out;
}

void FreeByteArray(unsigned char* p) {
    delete[] p;
}

unsigned int StringHash(const char* s) {
    if (!s) return 0;
    unsigned int h = 5381;
    for (; *s; ++s)
        h = ((h << 5) + h) + static_cast<unsigned char>(*s);
    return h;
}

// --- ABC helpers ---

int AbcPosition(char c) {
    if (c >= 'a' && c <= 'z') return static_cast<int>(c - 'a') + 1;
    if (c >= 'A' && c <= 'Z') return static_cast<int>(c - 'A') + 1;
    return 0;
}

int AbcScore(const char* s) {
    if (!s) return 0;
    int sum = 0;
    for (; *s; ++s)
        sum += AbcPosition(*s);
    return sum;
}

} // extern "C"
