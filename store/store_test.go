package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockGenerator struct{}

func (m *MockGenerator) NextID() (uint64, error) {
	return 123, nil
}

func TestCustomGenerator(t *testing.T) {
	assert.False(t, IsCustomGenerator())
	InitCustomGenerator(&MockGenerator{})
	assert.True(t, IsCustomGenerator())
}
