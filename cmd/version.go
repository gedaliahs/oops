package cmd

import (
	"strconv"
	"strings"
)

func compareVersions(a, b string) int {
	ap := versionParts(a)
	bp := versionParts(b)
	for i := 0; i < len(ap) || i < len(bp); i++ {
		av, bv := 0, 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func versionParts(version string) []int {
	version = strings.TrimPrefix(version, "v")
	fields := strings.Split(version, ".")
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimFunc(field, func(r rune) bool {
			return r < '0' || r > '9'
		})
		n, _ := strconv.Atoi(field)
		parts = append(parts, n)
	}
	return parts
}
