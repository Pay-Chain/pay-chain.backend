package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPaginationParams(t *testing.T) {
	p := GetPaginationParams(0, -1)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 0, p.Limit)

	p = GetPaginationParams(2, 20)
	assert.Equal(t, 2, p.Page)
	assert.Equal(t, 20, p.Limit)
}

func TestCalculateOffset(t *testing.T) {
	p := PaginationParams{Page: 1, Limit: 20}
	assert.Equal(t, 0, p.CalculateOffset())

	p = PaginationParams{Page: 3, Limit: 20}
	assert.Equal(t, 40, p.CalculateOffset())
}

func TestCalculateMeta(t *testing.T) {
	meta := CalculateMeta(100, 2, 20)
	assert.Equal(t, 2, meta.Page)
	assert.Equal(t, 20, meta.Limit)
	assert.Equal(t, int64(100), meta.TotalCount)
	assert.Equal(t, 5, meta.TotalPages)

	noLimit := CalculateMeta(15, 1, 0)
	assert.Equal(t, 1, noLimit.Page)
	assert.Equal(t, 15, noLimit.Limit)
	assert.Equal(t, 1, noLimit.TotalPages)
}
