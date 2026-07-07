package column

import (
	"fmt"
	"testing"
)

// BenchmarkCategoricalCodeOfHighCardinality exercises label resolution
// with 50K categories: the lazy lookup map keeps it O(1) (v0.7.1;
// the previous linear scan was O(k) per call).
func BenchmarkCategoricalCodeOfHighCardinality(b *testing.B) {
	c := highCardinality(b, 50_000)
	labels := make([]any, 64)
	for i := range labels {
		labels[i] = fmt.Sprintf("label-%06d", (i*777)%50_000)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if c.CodeOf(labels[i%len(labels)]) < 0 {
			b.Fatal("label not found")
		}
	}
}
