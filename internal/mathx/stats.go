// Package mathx contains the numerical primitives the brain leans on:
// random sampling for Bayesian bandits, logistic/sigmoid for probability
// models, and small linear-algebra helpers. Everything here is dependency-free
// and deterministic when seeded, so unit tests and the simulator are reproducible.
package mathx

import (
	"math"
	"math/rand"
)

// Sigmoid maps a real-valued logit to a probability in (0,1).
func Sigmoid(x float64) float64 {
	// Numerically stable form to avoid overflow for large |x|.
	if x >= 0 {
		z := math.Exp(-x)
		return 1.0 / (1.0 + z)
	}
	z := math.Exp(x)
	return z / (1.0 + z)
}

// Logit is the inverse of Sigmoid.
func Logit(p float64) float64 {
	p = Clamp(p, 1e-9, 1-1e-9)
	return math.Log(p / (1 - p))
}

// Clamp constrains x to [lo, hi].
func Clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// Dot computes the dot product of two equal-length vectors.
func Dot(a, b []float64) float64 {
	var s float64
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		s += a[i] * b[i]
	}
	return s
}

// SampleGamma draws from a Gamma(shape, scale) distribution using the
// Marsaglia-Tsang method. This is the building block for Beta sampling, which
// is the heart of Thompson sampling in the budget allocator.
func SampleGamma(rng *rand.Rand, shape, scale float64) float64 {
	if shape < 1 {
		// Boost low-shape draws (Marsaglia-Tsang correction).
		u := rng.Float64()
		return SampleGamma(rng, shape+1, scale) * math.Pow(u, 1.0/shape)
	}
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9*d)
	for {
		var x, v float64
		for {
			x = rng.NormFloat64()
			v = 1 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1-0.0331*x*x*x*x {
			return d * v * scale
		}
		if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v * scale
		}
	}
}

// SampleBeta draws from Beta(alpha, beta). Used to sample a plausible conversion
// rate for each ad "arm" given its observed successes/failures.
func SampleBeta(rng *rand.Rand, alpha, beta float64) float64 {
	if alpha <= 0 {
		alpha = 1e-3
	}
	if beta <= 0 {
		beta = 1e-3
	}
	x := SampleGamma(rng, alpha, 1.0)
	y := SampleGamma(rng, beta, 1.0)
	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

// BetaMean is the expected value of Beta(alpha, beta).
func BetaMean(alpha, beta float64) float64 {
	if alpha+beta == 0 {
		return 0.5
	}
	return alpha / (alpha + beta)
}

// SinHour and CosHour encode an hour-of-day (0..23) as a point on the unit
// circle. Using BOTH sin and cos removes the aliasing a lone sin term suffers
// (sin alone makes 3:00 and 9:00 — symmetric about the peak — indistinguishable).
func SinHour(h float64) float64 { return math.Sin(2 * math.Pi * h / 24.0) }
func CosHour(h float64) float64 { return math.Cos(2 * math.Pi * h / 24.0) }

// Sum returns the sum of a slice.
func Sum(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return s
}

// PoissonCDF returns P(X ≤ k) for X ~ Poisson(lambda), summed stably. Used by the
// guarantee controller to compute the probability a clinic falls short of its
// monthly target given the expected remaining arrivals.
func PoissonCDF(k int, lambda float64) float64 {
	if lambda <= 0 {
		if k >= 0 {
			return 1
		}
		return 0
	}
	if k < 0 {
		return 0
	}
	// term_i = e^-λ λ^i / i!, accumulated iteratively.
	term := math.Exp(-lambda)
	sum := term
	for i := 1; i <= k; i++ {
		term *= lambda / float64(i)
		sum += term
	}
	return Clamp(sum, 0, 1)
}

// Mean returns the arithmetic mean, or 0 for an empty slice.
func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	return Sum(xs) / float64(len(xs))
}
