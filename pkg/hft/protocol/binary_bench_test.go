package protocol

import (
	"testing"
)

// BenchmarkNewOrderMarshal benchmarks binary encoding
func BenchmarkNewOrderMarshal(b *testing.B) {
	msg := &NewOrderMessage{
		OrderID:  12345,
		Price:    5000000000,
		Quantity: 100,
		SymbolID: 1,
		Side:     SideBuy,
		Type:     OrderTypeLimit,
		TIF:      TIFGTC,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data := msg.MarshalBinary()
		PutBuffer(&data)
	}
}

// BenchmarkNewOrderUnmarshal benchmarks binary decoding
func BenchmarkNewOrderUnmarshal(b *testing.B) {
	msg := &NewOrderMessage{
		OrderID:  12345,
		Price:    5000000000,
		Quantity: 100,
		SymbolID: 1,
		Side:     SideBuy,
		Type:     OrderTypeLimit,
		TIF:      TIFGTC,
	}

	data := msg.MarshalBinary()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg2 := &NewOrderMessage{}
		_ = msg2.UnmarshalBinary(data)
	}

	PutBuffer(&data)
}

// BenchmarkNewOrderMarshalWithPool benchmarks with object pooling
func BenchmarkNewOrderMarshalWithPool(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := GetNewOrder()
		msg.OrderID = 12345
		msg.Price = 5000000000
		msg.Quantity = 100
		msg.SymbolID = 1
		msg.Side = SideBuy
		msg.Type = OrderTypeLimit
		msg.TIF = TIFGTC

		data := msg.MarshalBinary()
		PutBuffer(&data)
		PutNewOrder(msg)
	}
}

// BenchmarkNewOrderUnmarshalWithPool benchmarks with object pooling
func BenchmarkNewOrderUnmarshalWithPool(b *testing.B) {
	msg := GetNewOrder()
	msg.OrderID = 12345
	msg.Price = 5000000000
	msg.Quantity = 100
	msg.SymbolID = 1
	msg.Side = SideBuy
	msg.Type = OrderTypeLimit
	msg.TIF = TIFGTC

	data := msg.MarshalBinary()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg2 := GetNewOrder()
		_ = msg2.UnmarshalBinary(data)
		PutNewOrder(msg2)
	}

	PutBuffer(&data)
	PutNewOrder(msg)
}

// BenchmarkMarketDataMarshal benchmarks market data encoding
func BenchmarkMarketDataMarshal(b *testing.B) {
	msg := &MarketDataMessage{
		SymbolID:  1,
		BidPrice:  4999000000,
		BidQty:    100,
		AskPrice:  5001000000,
		AskQty:    50,
		LastPrice: 5000000000,
		LastQty:   75,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data := msg.MarshalBinary()
		PutBuffer(&data)
	}
}

// BenchmarkMarketDataUnmarshal benchmarks market data decoding
func BenchmarkMarketDataUnmarshal(b *testing.B) {
	msg := &MarketDataMessage{
		SymbolID:  1,
		BidPrice:  4999000000,
		BidQty:    100,
		AskPrice:  5001000000,
		AskQty:    50,
		LastPrice: 5000000000,
		LastQty:   75,
	}

	data := msg.MarshalBinary()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg2 := &MarketDataMessage{}
		_ = msg2.UnmarshalBinary(data)
	}

	PutBuffer(&data)
}

// BenchmarkBatchEncoder benchmarks batch encoding
func BenchmarkBatchEncoder(b *testing.B) {
	encoder := NewBatchEncoder(1024 * 100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := GetNewOrder()
		msg.OrderID = uint64(i)
		msg.Price = 5000000000
		msg.Quantity = 100
		msg.SymbolID = 1
		msg.Side = SideBuy
		msg.Type = OrderTypeLimit
		msg.TIF = TIFGTC

		_ = encoder.EncodeNewOrder(msg)
		PutNewOrder(msg)

		if i%100 == 0 {
			encoder.Reset()
		}
	}
}

// BenchmarkParallelMarshal benchmarks parallel encoding
func BenchmarkParallelMarshal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg := GetNewOrder()
			msg.OrderID = 12345
			msg.Price = 5000000000
			msg.Quantity = 100
			msg.SymbolID = 1
			msg.Side = SideBuy
			msg.Type = OrderTypeLimit
			msg.TIF = TIFGTC

			data := msg.MarshalBinary()
			PutBuffer(&data)
			PutNewOrder(msg)
		}
	})
}

// BenchmarkUnsafeMarshal benchmarks unsafe zero-copy marshaling
func BenchmarkUnsafeMarshal(b *testing.B) {
	msg := &NewOrderMessage{
		OrderID:  12345,
		Price:    5000000000,
		Quantity: 100,
		SymbolID: 1,
		Side:     SideBuy,
		Type:     OrderTypeLimit,
		TIF:      TIFGTC,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = UnsafeMarshalNewOrder(msg)
	}
}

// BenchmarkUnsafeUnmarshal benchmarks unsafe zero-copy unmarshaling
func BenchmarkUnsafeUnmarshal(b *testing.B) {
	msg := &NewOrderMessage{
		OrderID:  12345,
		Price:    5000000000,
		Quantity: 100,
		SymbolID: 1,
		Side:     SideBuy,
		Type:     OrderTypeLimit,
		TIF:      TIFGTC,
	}

	data := UnsafeMarshalNewOrder(msg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = UnsafeUnmarshalNewOrder(data)
	}
}
