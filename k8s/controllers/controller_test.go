package controllers

import (
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "config")
	defer func() {
		_ = os.RemoveAll(f.Name())
	}()
	assert.Nil(t, err)
	_, err = io.WriteString(f, "key: value")
	assert.Nil(t, err)

	mirror := &MirrorReconciler{ConfigFilepath: f.Name()}
	items, err := mirror.loadConfigItems()
	assert.Nil(t, err)
	assert.Equal(t, "value", items["key"])
}
