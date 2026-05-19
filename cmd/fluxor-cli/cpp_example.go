// Copyright (c) 2024-2028 Fluxor Framework
// All rights reserved.

package main

import (
	"fmt"

	"github.com/fluxorio/fluxor/base/bridge"
)

// runCppExample demonstrates Go calling C++ functions via base/private/cpp_bridge
func runCppExample() {
	fmt.Println("=== Go Calling C++ Example (base/bridge/cpp_bridge) ===\n")

	// Example 1: Simple arithmetic
	fmt.Println("1. Simple Arithmetic:")
	a, b := 42, 8
	sum := cpp_bridge.Add(a, b)
	fmt.Printf("   %d + %d = %d\n", a, b, sum)

	x, y := 3.14, 2.71
	product := cpp_bridge.Multiply(x, y)
	fmt.Printf("   %.2f * %.2f = %.2f\n\n", x, y, product)

	// Example 2: String operations
	fmt.Println("2. String Concatenation:")
	str1 := "Hello, "
	str2 := "C++ from Go!"
	concatenated := cpp_bridge.Concatenate(str1, str2)
	fmt.Printf("   \"%s\" + \"%s\" = \"%s\"\n\n", str1, str2, concatenated)

	// Example 3: Array operations
	fmt.Println("3. Array Sum:")
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	sumArray := cpp_bridge.SumArray(numbers)
	fmt.Printf("   Sum of %v = %d\n\n", numbers, sumArray)

	// Example 4: C++ class wrapper (Calculator)
	fmt.Println("4. C++ Calculator Class:")
	calc := cpp_bridge.NewCalculator()
	defer calc.Close()

	calc.SetValue(10.0)
	fmt.Printf("   Initial value: %.2f\n", calc.GetValue())

	result := calc.Calculate("add", 5.0)
	fmt.Printf("   After add(5): %.2f\n", result)

	result = calc.Calculate("multiply", 3.0)
	fmt.Printf("   After multiply(3): %.2f\n", result)

	result = calc.Calculate("subtract", 2.0)
	fmt.Printf("   After subtract(2): %.2f\n", result)

	result = calc.Calculate("power", 2.0)
	fmt.Printf("   After power(2): %.2f\n", result)

	result = calc.Calculate("divide", 4.0)
	fmt.Printf("   After divide(4): %.2f\n\n", result)

	// Example 5: C++20 features
	fmt.Println("5. C++20 Features:")

	fmt.Println("   a) String View Processing:")
	processed := cpp_bridge.ProcessStringView("hello c++20")
	fmt.Printf("      Input: \"hello c++20\"\n")
	fmt.Printf("      Output: %s\n\n", processed)

	fmt.Println("   b) Finding Maximum (C++20 algorithms):")
	maxArr := []int{45, 12, 78, 23, 91, 56, 34, 67}
	maxVal := cpp_bridge.FindMax(maxArr)
	fmt.Printf("      Array: %v\n", maxArr)
	fmt.Printf("      Maximum: %d\n\n", maxVal)

	fmt.Println("   c) Optional Values (C++20 std::optional):")
	opt1 := cpp_bridge.OptionalValue(42.5, true)
	fmt.Printf("      Optional(42.5, true): %s\n", opt1)
	opt2 := cpp_bridge.OptionalValue(0.0, false)
	fmt.Printf("      Optional(0.0, false): %s\n", opt2)
	opt3 := cpp_bridge.OptionalValue(-5.0, true)
	fmt.Printf("      Optional(-5.0, true): %s\n\n", opt3)

	fmt.Println("=== Example Complete ===")
}
