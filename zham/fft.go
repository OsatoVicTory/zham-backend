package zham

import (
	"math"
)

func FFT(input []float64) []complex128 {
	complexArray := make([]complex128, len(input))
	for i, v := range input {
		complexArray[i] = complex(v, 0)
	}

	fftRes := make([]complex128, len(complexArray))
	copy(fftRes, complexArray)
	return recursiveFFT(fftRes)
}

func recursiveFFT(complexArray []complex128) []complex128 {
	N := len(complexArray)
	if N <= 1 {
		return complexArray
	}

	even := make([]complex128, N/2)
	odd := make([]complex128, N/2)
	for i := 0; i < N/2; i++ {
		even[i] = complexArray[2*i]
		odd[i] = complexArray[2*i+1]
	}

	even = recursiveFFT(even)
	odd = recursiveFFT(odd)

	fftRes := make([]complex128, N)
	for k := 0; k < N/2; k++ {
		angle := (-2 * math.Pi * float64(k)) / float64(N)
		t := complex(math.Cos(angle), math.Sin(angle))
		fftRes[k] = even[k] + t*odd[k]
		fftRes[k+N/2] = even[k] - t*odd[k]
	}

	return fftRes
}
