/*
   Copyright Mycophonic.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package resample

import "math"

// besselRangeBoundary is the threshold between the two rational polynomial
// approximation ranges in besselI0 (Blair & Edwards, AECL-4928).
const besselRangeBoundary = 15.0

// besselI0 computes the Modified Bessel function of the first kind, order 0.
// This is a port of ffmpeg's av_bessel_i0, which uses the Blair & Edwards (1974)
// minimax rational polynomial approximation (Chalk River Report AECL-4928).
func besselI0(val float64) float64 {
	if val == 0 {
		return 1
	}

	val = math.Abs(val)

	if val <= besselRangeBoundary {
		// Range [0, 15]: rational polynomial in y = val*val.
		ySq := val * val

		return evalPoly(besselP1[:], ySq) / evalPoly(besselQ1[:], ySq)
	}

	// Range (15, +inf): asymptotic expansion.
	ySq := 1/val - 1/besselRangeBoundary
	ratio := evalPoly(besselP2[:], ySq) / evalPoly(besselQ2[:], ySq)

	return math.Exp(val) / math.Sqrt(val) * ratio
}

// evalPoly evaluates a polynomial using Horner's method: p[0] + p[1]*x + p[2]*x^2 + ...
func evalPoly(coeffs []float64, point float64) float64 {
	result := coeffs[len(coeffs)-1]

	for idx := len(coeffs) - 2; idx >= 0; idx-- {
		result = result*point + coeffs[idx]
	}

	return result
}

// Blair & Edwards coefficients for besselI0, range [0, 15].
//
//nolint:gochecknoglobals
var besselP1 = [15]float64{
	-2.2335582639474375249e+15,
	-5.5050369673018427753e+14,
	-3.2940087627407749166e+13,
	-8.4925101247114157499e+11,
	-1.1912746104985237192e+10,
	-1.0313066708737980747e+08,
	-5.9545626019847898221e+05,
	-2.4125195876041896775e+03,
	-7.0935347449210549190e+00,
	-1.5453977791786851041e-02,
	-2.5172644670688975051e-05,
	-3.0517226450451067446e-08,
	-2.6843448573468483278e-11,
	-1.5982226675653184646e-14,
	-5.2487866627945699800e-18,
}

//nolint:gochecknoglobals
var besselQ1 = [6]float64{
	-2.2335582639474375245e+15,
	7.8858692566751002988e+12,
	-1.2207067397808979846e+10,
	1.0377081058062166144e+07,
	-4.8527560179962773045e+03,
	1.0,
}

// Blair & Edwards coefficients for besselI0, range (15, +inf).
//
//nolint:gochecknoglobals
var besselP2 = [7]float64{
	-2.2210262233306573296e-04,
	1.3067392038106924055e-02,
	-4.4700805721174453923e-01,
	5.5674518371240761397e+00,
	-2.3517945679239481621e+01,
	3.1611322818701131207e+01,
	-9.6090021968656180000e+00,
}

//nolint:gochecknoglobals
var besselQ2 = [8]float64{
	-5.5194330231005480228e-04,
	3.2547697594819615062e-02,
	-1.1151759188741312645e+00,
	1.3982595353892851542e+01,
	-6.0228002066743340583e+01,
	8.5539563258012929600e+01,
	-3.1446690275135491500e+01,
	1.0,
}

// kaiserWindowScale is the scaling factor for the Kaiser window argument (2.0).
const kaiserWindowScale = 2.0

// buildFilterBank constructs a Kaiser-windowed sinc polyphase filter bank
// matching ffmpeg's swr build_filter. Returns [phases+1][filterLen] float64
// coefficients normalized so each phase sums to ~1.0.
// The extra phase at index phases is for linear interpolation wrap-around.
//
// This implementation matches two key swr optimizations:
//   - For factor==1.0 (upsampling), uses a precomputed sin lookup with sign
//     flipping instead of computing sin(x)/x per tap.
//   - For even phase counts, only computes the first half of phases and mirrors
//     the rest: filter[N-ph][filterLen-1-i] = filter[ph][i].
//
// Normalization uses phase 0's coefficient sum as the divisor for all phases,
// matching swr's behavior.
func buildFilterBank(factor float64, filterLen, phases int, beta float64) [][]float64 {
	bank := make([][]float64, phases+1)

	for idx := range bank {
		bank[idx] = make([]float64, filterLen)
	}

	// Integer division matches swr: center = (tap_count-1)/2.
	center := (filterLen - 1) / 2
	centerF := float64(center)

	// Number of phases to compute explicitly.
	// For even phase_count, compute first half + 1 and mirror the rest.
	phaseHalf := phases
	if phases%2 == 0 {
		phaseHalf = phases/2 + 1
	}

	// Precompute sin lookup for factor==1.0 (matches swr's sin_lut optimization).
	var sinLut []float64

	const unitFactor = 1.0

	if factor == unitFactor {
		sinLut = make([]float64, phaseHalf)
		// swr: sin(M_PI * ph / phase_count) * (center & 1 ? 1 : -1)
		signFactor := unitFactor
		if center%2 == 0 {
			signFactor = -unitFactor
		}

		for phase := range phaseHalf {
			sinLut[phase] = math.Sin(math.Pi*float64(phase)/float64(phases)) * signFactor
		}
	}

	// Build first half of phases.
	for phase := range phaseHalf {
		phaseOffset := float64(phase) / float64(phases)

		// Initialize sin value for this phase (flips sign per tap).
		var sinVal float64
		if sinLut != nil {
			sinVal = sinLut[phase]
		}

		for tap := range filterLen {
			sincArg := (float64(tap) - centerF) - phaseOffset
			sincVal := math.Pi * sincArg * factor

			var coeff float64

			switch {
			case sincVal == 0:
				coeff = 1
			case sinLut != nil:
				// factor==1.0 path: use precomputed sin with sign flipping.
				coeff = sinVal / sincVal
			default:
				coeff = math.Sin(sincVal) / sincVal
			}

			// Kaiser window: w = 2*x/(factor*tap_count*PI) where x = PI*sincArg*factor.
			// Simplifies to w = 2*sincArg/filterLen (factor and PI cancel).
			windowArg := kaiserWindowScale * math.Abs(sincArg) / float64(filterLen)
			windowSq := max(1-windowArg*windowArg, 0)

			coeff *= besselI0(beta * math.Sqrt(windowSq))
			bank[phase][tap] = coeff

			// Flip sin sign for next tap (matches swr's s = -s).
			sinVal = -sinVal
		}

		// Mirror: filter[phases-phase][filterLen-1-i] = filter[phase][i].
		if phases%2 == 0 {
			mirrorPhase := phases - phase
			if mirrorPhase < phases {
				for tap := range filterLen {
					bank[mirrorPhase][filterLen-1-tap] = bank[phase][tap]
				}
			}
		}
	}

	// Normalize by phase 0 sum (matches swr: norm accumulated only for ph==0).
	var phase0Sum float64
	for tap := range filterLen {
		phase0Sum += bank[0][tap]
	}

	for phase := range phases {
		for tap := range filterLen {
			bank[phase][tap] /= phase0Sum
		}
	}

	// Phase [phases] wraps to phase [0] for linear interpolation.
	copy(bank[phases], bank[0])

	return bank
}
