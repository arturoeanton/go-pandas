package ndarray

import (
	"math/rand"
	"sync"
)

// rng is the package random source. rand.Rand is not safe for concurrent
// use, so every access goes through rngMu.
var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(rand.Int63()))
)

// Seed makes the random constructors deterministic, like np.random.seed.
func Seed(seed int64) {
	rngMu.Lock()
	defer rngMu.Unlock()
	rng = rand.New(rand.NewSource(seed))
}

// Rand returns an array of uniform random samples in [0, 1).
func Rand(shape ...int) *NDArray {
	a := Zeros(shape...)
	d := a.data.([]float64)
	rngMu.Lock()
	defer rngMu.Unlock()
	for i := range d {
		d[i] = rng.Float64()
	}
	return a
}

// Randn returns an array of standard normal samples.
func Randn(shape ...int) *NDArray {
	a := Zeros(shape...)
	d := a.data.([]float64)
	rngMu.Lock()
	defer rngMu.Unlock()
	for i := range d {
		d[i] = rng.NormFloat64()
	}
	return a
}

// RandInt returns an array of uniform random integers in [low, high).
func RandInt(low, high int, shape ...int) *NDArray {
	a := Zeros(shape...)
	d := a.data.([]float64)
	span := high - low
	rngMu.Lock()
	defer rngMu.Unlock()
	for i := range d {
		d[i] = float64(low + rng.Intn(span))
	}
	return a
}
