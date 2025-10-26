package container

import (
	"log"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetImageArchitectures requires a running Docker daemon.
func TestGetImageArchitectures(t *testing.T) {

	// Pull a known multi-arch image for testing
	// busybox is available for amd64, arm64, and others.
	InitDockerContainerFactory()
	err := pullImage("busybox:latest")
	assert.NoError(t, err)
	t.Run("multi-arch image", func(t *testing.T) {
		archs, err := cf.GetImageArchitectures("busybox:latest")
		assert.NoError(t, err)
		assert.Contains(t, archs, "amd64")
		assert.Contains(t, archs, "arm64")
		assert.Equal(t, len(archs), 2)
	})

	t.Run("single-arch image", func(t *testing.T) {
		// Let's use a specific arch image from the registry
		image := "amd64/hello-world"
		archs, err := cf.GetImageArchitectures(image)
		assert.NoError(t, err)
		assert.Equal(t, []string{"amd64"}, archs)
	})

	t.Run("non-existent image", func(t *testing.T) {
		_, err := cf.GetImageArchitectures("non-existent-image-serverledge-test-multi-arch:latest")
		assert.Error(t, err)
	})

	t.Run("cached-etcd image", func(t *testing.T) {

		// This test is time-sensitive, so it's not ideal. The best thing here is to also check the log, where wc
		// can be 100% sure that the second invocation results in a cache hit.

		start := time.Now()
		archs, err := cf.GetImageArchitectures("memcached:latest")
		noCacheElapsed := time.Since(start)
		assert.NoError(t, err)
		assert.Contains(t, archs, "amd64")
		assert.Contains(t, archs, "arm64")
		assert.Equal(t, len(archs), 2)

		// Now should be in cache
		start = time.Now() // reset timer
		archs, err = cf.GetImageArchitectures("memcached:latest")
		cacheElapsed := time.Since(start)
		assert.NoError(t, err)
		assert.Contains(t, archs, "amd64")
		assert.Contains(t, archs, "arm64")
		assert.Equal(t, len(archs), 2)
		// The cached call should be significantly faster
		assert.Less(t, cacheElapsed, noCacheElapsed, "Cached call should be faster than non-cached call")

	})
}

func pullImage(image string) error {
	cmd := exec.Command("docker", "pull", image)

	// Run esegue il comando e attende la fine.
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed pulling image: %v\n", err)
		return err
	}
	return nil

}
