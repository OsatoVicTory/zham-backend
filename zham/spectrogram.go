package zham

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
	"sort"
	"zham-app/models"
)

const (
	dspRatio        = 4
	freqBinSize     = 2048 //1024
	freqBinSizeHalf = freqBinSize / 2
	maxFreq         = 6000.0
	minFreq         = 100.0
	hopSize         = 64 //32
)

func LowPassFilter(cutoffFrequency, sampleRate float64, input []float64) []float64 {
	rc := 1.0 / (2 * math.Pi * cutoffFrequency)
	dt := 1.0 / sampleRate
	alpha := dt / (rc + dt)

	filteredSignal := make([]float64, len(input))

	for i, x := range input {
		if i == 0 {
			filteredSignal[i] = x * alpha
		} else {
			filteredSignal[i] = alpha*x + (1-alpha)*filteredSignal[i-1]
		}
	}
	return filteredSignal
}

func HighPassFilter(cutoffFrequency, sampleRate float64, input []float64) []float64 {
	// corrected to the correct highpass filter algorithm
	rc := 1.0 / (2 * math.Pi * cutoffFrequency)
	dt := 1.0 / sampleRate
	alpha := rc / (rc + dt)

	filteredSignal := make([]float64, len(input))

	for i, x := range input {
		if i == 0 {
			filteredSignal[i] = x * alpha
		} else {
			filteredSignal[i] = alpha*filteredSignal[i-1] + (x - input[i-1])
		}
	}
	return filteredSignal
}

func DownSample(input []float64, sampleRate int) ([]float64, float64, error) {
	if dspRatio <= 0 {
		return nil, 0.0, errors.New("ratio must be positive")
	}

	var resampled []float64
	for i := 0; i < len(input); i += dspRatio {
		end := min(i+dspRatio, len(input))

		sum := 0.0
		for j := i; j < end; j++ {
			sum += input[j]
		}
		avg := sum / float64(end-i)
		resampled = append(resampled, avg)
	}

	newSampleRate := float64(sampleRate) / float64(dspRatio)
	return resampled, newSampleRate, nil
}

func Spectrogram(sample []float64, sampleRate int) ([][]float64, []float64, error) {
	// using wav sample rate as 48KHz, and we will downsample(48kHz / 4) to 12kHz, so max freq is 12Khz / 2 = 6KHz
	filteredSignal := LowPassFilter(maxFreq, float64(sampleRate), sample)

	samples, newSampleRate, err := DownSample(filteredSignal, sampleRate)
	if err != nil {
		return nil, nil, fmt.Errorf("could not downsample audio sample: %v", err)
	}

	numWindows := int((len(samples) - freqBinSize) / hopSize)

	spectrogramMags := make([][]float64, numWindows)
	time := make([]float64, numWindows)
	window := make([]float64, freqBinSize)

	for i := range window {
		window[i] = 0.54 - (0.46 * math.Cos(2*math.Pi*float64(i)/(float64(freqBinSize)-1)))
	}

	for i := range time {
		start := i * hopSize
		end := min(start+freqBinSize, len(samples))

		bin := make([]float64, freqBinSize)
		copy(bin, samples[start:end])

		for j := range window {
			bin[j] *= window[j]
		}

		spectrogramBin := FFT(bin)

		binMags := make([]float64, freqBinSizeHalf)
		for fi, freq := range spectrogramBin {
			if fi < freqBinSizeHalf {
				mag := cmplx.Abs(freq)
				scaledMag := 20.0 * math.Log10(max(mag, 1e-10)) // scaled to dB
				binMags[fi] = scaledMag
			} else {
				break
			}
		}

		spectrogramMags[i] = binMags
		time[i] = float64(start) / newSampleRate
	}

	return spectrogramMags, time, nil
}

type Bands struct{ min, max int32 }

func getLogBands(max_freq float64, min_freq float64, num_bands int, L float64) []Bands {
	if min_freq == 0 {
		min_freq = 1
	}
	if num_bands == 0 {
		num_bands = 1
	}
	div := max_freq / min_freq

	float_num_bands := float64(num_bands)
	bands := make([]Bands, 0)
	var prev int32
	// logarithmic spacing on n=30 bands
	for i := 0; i < num_bands; i++ {
		freq := min_freq * (math.Pow(div, float64(i+1)/float_num_bands))
		nxt := int32(min(L, (freq/max_freq)*L))
		if i > 0 {
			bands = append(bands, Bands{min: prev, max: nxt})
		}
		prev = nxt
	}
	return bands
}

func GetPeaks(spectrogram [][]float64, time []float64, dist_time int, dist_freq int, coeff int) []models.Peak {

	type maxStruct struct {
		maxFreqAmplitude float64
		Freq             int32
		Time             float64
	}

	N := len(spectrogram)
	K := len(spectrogram[0])

	bands := getLogBands(6000.0, 300.0, 30, float64(K))
	E := make([][]maxStruct, N)
	var peaks []models.Peak

	for i, bin := range spectrogram {
		bandsEnergies := make([]maxStruct, len(bin))
		for bi, band := range bands {
			var maxMag maxStruct
			for pos := band.min; pos < band.max; pos++ {
				mag := bin[pos]
				if mag > maxMag.maxFreqAmplitude {
					maxMag = maxStruct{mag, pos, time[i]}
				}
			}
			bandsEnergies[bi] = maxMag
		}
		E[i] = bandsEnergies
	}

	K = len(E[0])
	C := make([][]bool, N)

	for s := range C {
		C[s] = make([]bool, K)
	}

	sum := 0.0
	num := 0

	for n, bin := range E {
		for k := range bin {
			mag := bin[k].maxFreqAmplitude
			startN := max(0, n-dist_time)
			endN := min(N, n+dist_time)

			startK := max(0, k-dist_freq)
			endK := min(K, k+dist_freq)

			ok := true

			for i := startN; i < endN; i++ {
				for j := startK; j < endK; j++ {
					if i != n && j != k && E[i][j].maxFreqAmplitude > mag {
						ok = false
						break
					}
				}
				if !ok {
					break
				}
			}

			if ok {
				C[n][k] = true
				sum += mag
				num++
			}
		}
	}

	mean := sum / float64(num)
	stdDev := 0.0
	for n, bin := range C {
		for k := range bin {
			if bin[k] {
				stdDev += math.Pow(E[n][k].maxFreqAmplitude-mean, 2)
			}
		}
	}
	stdDev = math.Sqrt(stdDev / float64(num))
	avg := mean + stdDev

	for n, bin := range C {
		for k := range bin {
			if bin[k] {
				ref := E[n][k]
				if ref.maxFreqAmplitude >= avg {
					peaks = append(peaks, models.Peak{Time: ref.Time, Freq: ref.Freq})
				}
			}
		}
	}

	sort.Slice(peaks, func(i, j int) bool {
		if peaks[i].Time != peaks[j].Time {
			return peaks[i].Time < peaks[j].Time
		}
		return peaks[i].Freq < peaks[j].Freq // if same time, sort in ascending freq
	})

	return peaks
}

func ExtractPeaks(spectrogram [][]complex128, time []float64, coeff float64) []models.Peak {
	if len(spectrogram) < 1 {
		return []models.Peak{}
	}

	// type maxStruct struct {
	// 	maxFreqAmplitude float64
	// 	Freq             int32
	// 	Time             float64
	// }

	var peaks []models.Peak
	bands := []struct{ min, max int32 }{{0, 10}, {10, 20}, {20, 64}, {64, 128}, {128, 192}, {192, 360}, {360, 512}}
	// bands := []struct{ min, max int32 }{{0, 8}, {8, 32}, {32, 64}, {64, 250}, {250, 500}, {500, 750}, {750, 1000}, {1000, 2000}}

	// var maxes []maxStruct

	for i, bin := range spectrogram {
		maxs := 0.0
		for _, band := range bands {
			var maxMag float64
			for freq := band.min; freq < band.max; freq++ {
				magnitude := cmplx.Abs(bin[freq])
				if magnitude > maxMag {
					maxMag = magnitude
				}
			}
			maxs += maxMag
		}

		max_mean := maxs / float64(len(bands))
		maxs_mean := 0.8 * max_mean
		for _, band := range bands {
			for freq := band.min; freq < band.max; freq++ {
				magnitude := cmplx.Abs(bin[freq])
				if magnitude > maxs_mean {
					// maxes = append(maxes, maxStruct{maxFreqAmplitude: magnitude, Freq: freq, Time: time[i]})
					peaks = append(peaks, models.Peak{Time: time[i], Freq: freq})
				}
			}
		}
	}

	// A := 0.8 // 0.6

	// var magSums float64
	// for _, max := range maxes {
	// 	magSums += max.maxFreqAmplitude
	// }

	// mean := magSums / float64(len(maxes))

	// fmt.Println("mean", mean)

	// avgVal := coeff * mean

	// for _, max := range maxes {
	// 	if max.maxFreqAmplitude >= avgVal {
	// 		peaks = append(peaks, models.Peak{Time: max.Time, Freq: max.Freq})
	// 	}
	// }

	sort.Slice(peaks, func(i, j int) bool {
		if peaks[i].Time != peaks[j].Time {
			return peaks[i].Time < peaks[j].Time
		}
		return peaks[i].Freq < peaks[j].Freq // if same time, sort in ascending freq
	})

	return peaks

}
