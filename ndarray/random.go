package ndarray

import "math/rand"

// rng is the package random source. Seed replaces it for reproducibility.
var rng = rand.New(rand.NewSource(rand.Int63()))

// Seed makes the random constructors deterministic, like np.random.seed.
func Seed(seed int64) { rng = rand.New(rand.NewSource(seed)) }

// Rand returns an array of uniform random samples in [0, 1).
func Rand(shape ...int) *NDArray {
	a := Zeros(shape...)
	for i := range a.data {
		a.data[i] = rng.Float64()
	}
	return a
}

// Randn returns an array of standard normal samples.
func Randn(shape ...int) *NDArray {
	a := Zeros(shape...)
	for i := range a.data {
		a.data[i] = rng.NormFloat64()
	}
	return a
}

// RandInt returns an array of uniform random integers in [low, high).
func RandInt(low, high int, shape ...int) *NDArray {
	a := Zeros(shape...)
	span := high - low
	for i := range a.data {
		a.data[i] = float64(low + rng.Intn(span))
	}
	return a
}
