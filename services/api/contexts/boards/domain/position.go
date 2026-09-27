package domain

// Positions are fractional index keys, ported from rocicorp's
// fractional-indexing (https://github.com/rocicorp/fractional-indexing, CC0).
// A key is an "integer part" (a head letter giving its length, then base-62
// digits) followed by an optional fraction that never ends in '0'. Keys
// compare as plain byte strings, and a new key can always be made between
// any two, so moving a task rewrites only that task's key.
//
// Keys must be stored and sorted by byte value (Postgres COLLATE "C").

import (
	"errors"
	"strings"
)

const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// smallestInteger is the one integer part with no key below it.
const smallestInteger = "A00000000000000000000000000"

var ErrInvalidPosition = errors.New("invalid position")

// KeyBetween returns a key strictly between a and b. An empty a means "no
// lower bound", an empty b "no upper bound"; both empty gives the first key.
func KeyBetween(a, b string) (string, error) {
	if a != "" {
		if err := validateKey(a); err != nil {
			return "", err
		}
	}
	if b != "" {
		if err := validateKey(b); err != nil {
			return "", err
		}
	}
	if a != "" && b != "" && a >= b {
		return "", ErrInvalidPosition
	}

	switch {
	case a == "" && b == "":
		return "a0", nil
	case a == "":
		ib := integerPart(b)
		fb := b[len(ib):]
		if ib == smallestInteger {
			m, err := midpoint("", fb, true)
			return ib + m, err
		}
		if ib < b {
			return ib, nil
		}
		res, ok := decrementInteger(ib)
		if !ok {
			return "", ErrInvalidPosition
		}
		return res, nil
	case b == "":
		ia := integerPart(a)
		fa := a[len(ia):]
		if i, ok := incrementInteger(ia); ok {
			return i, nil
		}
		m, err := midpoint(fa, "", false)
		return ia + m, err
	}

	ia := integerPart(a)
	fa := a[len(ia):]
	ib := integerPart(b)
	fb := b[len(ib):]
	if ia == ib {
		m, err := midpoint(fa, fb, true)
		return ia + m, err
	}
	i, ok := incrementInteger(ia)
	if !ok {
		return "", ErrInvalidPosition
	}
	if i < b {
		return i, nil
	}
	m, err := midpoint(fa, "", false)
	return ia + m, err
}

// midpoint returns a fraction strictly between a and b. hasB says whether b
// is a bound at all (an empty b with hasB is the empty fraction).
func midpoint(a, b string, hasB bool) (string, error) {
	if hasB && a >= b && b != "" {
		return "", ErrInvalidPosition
	}
	if strings.HasSuffix(a, "0") || (hasB && strings.HasSuffix(b, "0")) {
		return "", ErrInvalidPosition
	}
	if hasB && b != "" {
		// Shared prefix, padding a with zeros.
		n := 0
		for n < len(b) && digitAt(a, n) == b[n] {
			n++
		}
		if n > 0 {
			rest := ""
			if n < len(a) {
				rest = a[n:]
			}
			m, err := midpoint(rest, b[n:], true)
			return b[:n] + m, err
		}
	}
	digitA := 0
	if a != "" {
		digitA = strings.IndexByte(digits, a[0])
	}
	digitB := len(digits)
	if hasB && b != "" {
		digitB = strings.IndexByte(digits, b[0])
	}
	if digitB-digitA > 1 {
		return string(digits[(digitA+digitB+1)/2]), nil
	}
	if hasB && len(b) > 1 {
		return b[:1], nil
	}
	rest := ""
	if len(a) > 1 {
		rest = a[1:]
	}
	m, err := midpoint(rest, "", false)
	return string(digits[digitA]) + m, err
}

func digitAt(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return '0'
}

func integerLength(head byte) (int, bool) {
	switch {
	case head >= 'a' && head <= 'z':
		return int(head-'a') + 2, true
	case head >= 'A' && head <= 'Z':
		return int('Z'-head) + 2, true
	}
	return 0, false
}

func integerPart(key string) string {
	n, _ := integerLength(key[0])
	return key[:n]
}

func validateKey(key string) error {
	if key == smallestInteger {
		return ErrInvalidPosition
	}
	n, ok := integerLength(key[0])
	if !ok || n > len(key) {
		return ErrInvalidPosition
	}
	for i := 1; i < len(key); i++ {
		if strings.IndexByte(digits, key[i]) < 0 {
			return ErrInvalidPosition
		}
	}
	if strings.HasSuffix(key[n:], "0") {
		return ErrInvalidPosition
	}
	return nil
}

func incrementInteger(x string) (string, bool) {
	head := x[0]
	digs := []byte(x[1:])
	carry := true
	for i := len(digs) - 1; carry && i >= 0; i-- {
		d := strings.IndexByte(digits, digs[i]) + 1
		if d == len(digits) {
			digs[i] = '0'
		} else {
			digs[i] = digits[d]
			carry = false
		}
	}
	if !carry {
		return string(head) + string(digs), true
	}
	if head == 'Z' {
		return "a0", true
	}
	if head == 'z' {
		return "", false
	}
	h := head + 1
	if h > 'a' {
		digs = append(digs, '0')
	} else {
		digs = digs[:len(digs)-1]
	}
	return string(h) + string(digs), true
}

func decrementInteger(x string) (string, bool) {
	head := x[0]
	digs := []byte(x[1:])
	borrow := true
	for i := len(digs) - 1; borrow && i >= 0; i-- {
		d := strings.IndexByte(digits, digs[i]) - 1
		if d == -1 {
			digs[i] = digits[len(digits)-1]
		} else {
			digs[i] = digits[d]
			borrow = false
		}
	}
	if !borrow {
		return string(head) + string(digs), true
	}
	if head == 'a' {
		return "Z" + string(digits[len(digits)-1]), true
	}
	if head == 'A' {
		return "", false
	}
	h := head - 1
	if h < 'Z' {
		digs = append(digs, digits[len(digits)-1])
	} else {
		digs = digs[:len(digs)-1]
	}
	return string(h) + string(digs), true
}
