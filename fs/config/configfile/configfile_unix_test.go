//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package configfile

import (
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/rclone/rclone/fs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFromFIFO(t *testing.T) *Storage {
	t.Helper()
	fifoPath := t.TempDir() + "/rclone.conf.fifo"
	require.NoError(t, syscall.Mkfifo(fifoPath, 0600))

	old := config.GetConfigPath()
	require.NoError(t, config.SetConfigPath(fifoPath))
	t.Cleanup(func() {
		assert.NoError(t, config.SetConfigPath(old))
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		f, err := os.OpenFile(fifoPath, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer f.Close()
		_, _ = f.Write([]byte(configData))
	}()

	data := &Storage{}
	require.NoError(t, data.Load())
	wg.Wait()
	return data
}

func TestConfigFileFIFO(t *testing.T) {
	data := loadFromFIFO(t)

	assert.Equal(t, []string{"one", "two", "three"}, data.GetSectionList())

	value, ok := data.GetValue("one", "type")
	assert.True(t, ok)
	assert.Equal(t, "number1", value)

	value, ok = data.GetValue("two", "fruit")
	assert.True(t, ok)
	assert.Equal(t, "apple", value)

	value, ok = data.GetValue("three", "fruit")
	assert.True(t, ok)
	assert.Equal(t, "banana", value)
}

func TestConfigFileFIFOSaveIsRejected(t *testing.T) {
	data := loadFromFIFO(t)

	err := data.Save()
	require.Error(t, err)

	fi, statErr := os.Stat(config.GetConfigPath())
	require.NoError(t, statErr)
	assert.True(t, fi.Mode()&os.ModeNamedPipe != 0)
}
