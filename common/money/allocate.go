// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package money

// Allocate distributes m across the given ratio parts, handling any remainder
// so that the sum of the result equals m exactly. Each part must be >= 0 and at
// least one part must be > 0.
//
// Remainder units are assigned by largest-remainder order using integer
// remainders only (no float64), which keeps allocation fair for large amounts.
//
// Example: allocating 100 cents across [1, 1, 1] yields [34, 33, 33].
func (m Money) Allocate(parts []int) ([]Money, error) {
	if len(parts) == 0 {
		return nil, ErrNoAllocationParts
	}
	total := 0
	for _, p := range parts {
		if p < 0 {
			return nil, ErrNegativeAllocationPart
		}
		total += p
	}
	if total == 0 {
		return nil, ErrNoAllocationParts
	}

	result := make([]Money, len(parts))
	allocated := int64(0)
	remainders := make([]int64, len(parts))
	total64 := int64(total)

	for i, p := range parts {
		num, err := mulInt64(m.amount, int64(p))
		if err != nil {
			return nil, err
		}
		share := num / total64
		remainders[i] = num - share*total64 // integer remainder numerator
		if remainders[i] < 0 {
			remainders[i] = -remainders[i]
		}
		result[i] = Money{amount: share, currency: m.currency}
		allocated += share
	}

	remainder := m.amount - allocated
	if remainder == 0 {
		return result, nil
	}

	// Order indices by descending integer remainder; ties keep lower index first
	// (stable insertion sort).
	type idxRem struct {
		i   int
		rem int64
	}
	order := make([]idxRem, len(parts))
	for i := range parts {
		order[i] = idxRem{i: i, rem: remainders[i]}
	}
	for i := 1; i < len(order); i++ {
		for j := i; j > 0; j-- {
			if order[j].rem > order[j-1].rem ||
				(order[j].rem == order[j-1].rem && order[j].i < order[j-1].i) {
				order[j], order[j-1] = order[j-1], order[j]
			} else {
				break
			}
		}
	}

	step := int64(1)
	if remainder < 0 {
		step = -1
	}
	for remainder != 0 {
		for _, o := range order {
			if remainder == 0 {
				break
			}
			next, err := addInt64(result[o.i].amount, step)
			if err != nil {
				return nil, err
			}
			result[o.i].amount = next
			remainder -= step
		}
	}
	return result, nil
}
