package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFileDagReader_ReadPathsFromDir(t *testing.T) {
	file := &FileDagReader{}
	paths, err := file.ReadPathsFromDir("./tests")
	assert.NoError(t, err)
	wantPaths := []string{
		"tests\\sub-tests\\subtest.yaml",
		"tests\\testdag.yaml",
		"tests\\testdag2.yml",
	}
	assert.Equal(t, wantPaths, paths)
}
