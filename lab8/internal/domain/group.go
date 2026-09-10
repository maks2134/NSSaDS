package domain

import "math/rand"

// AssignColors builds a disjoint partition of P world ranks into K groups.
// If K <= P: K non-empty groups with random positive sizes summing to P.
// If K > P: P singleton groups (colors 0..P-1); colors K-P..K-1 unused.
// Returns color[rank] in 0..K-1 (or undefined for unused colors when K > P).
func AssignColors(p, k int, seed int64) []int32 {
	colors := make([]int32, p)
	if p <= 0 || k <= 0 {
		return colors
	}
	rng := rand.New(rand.NewSource(seed))

	if k >= p {
		for r := 0; r < p; r++ {
			colors[r] = int32(r)
		}
		return colors
	}

	sizes := randomPositiveSizes(p, k, rng)
	ranks := make([]int, p)
	for i := range ranks {
		ranks[i] = i
	}
	rng.Shuffle(p, func(i, j int) { ranks[i], ranks[j] = ranks[j], ranks[i] })

	idx := 0
	for color, sz := range sizes {
		for i := 0; i < sz; i++ {
			colors[ranks[idx]] = int32(color)
			idx++
		}
	}
	return colors
}

// randomPositiveSizes returns k positive integers that sum to p.
func randomPositiveSizes(p, k int, rng *rand.Rand) []int {
	sizes := make([]int, k)
	for i := range sizes {
		sizes[i] = 1
	}
	remaining := p - k
	for remaining > 0 {
		sizes[rng.Intn(k)]++
		remaining--
	}
	return sizes
}

// GroupMembers returns world ranks that have the given color, sorted ascending.
func GroupMembers(colors []int32, color int32) []int {
	var members []int
	for r, c := range colors {
		if c == color {
			members = append(members, r)
		}
	}
	return members
}

// MaxColor returns the largest color value present in colors, or -1 if empty.
func MaxColor(colors []int32) int {
	if len(colors) == 0 {
		return -1
	}
	m := int(colors[0])
	for _, c := range colors[1:] {
		if int(c) > m {
			m = int(c)
		}
	}
	return m
}
