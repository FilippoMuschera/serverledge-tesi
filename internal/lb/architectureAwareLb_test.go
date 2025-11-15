package lb

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/serverledge-faas/serverledge/internal/cache"
	"github.com/serverledge-faas/serverledge/internal/container"
	"github.com/serverledge-faas/serverledge/internal/function"
	"github.com/stretchr/testify/assert"
)

func newTarget(name string, arch string) *middleware.ProxyTarget {
	return &middleware.ProxyTarget{
		Name: name,
		URL:  &url.URL{Host: name},
		Meta: echo.Map{"arch": arch},
	}
}

func TestNewArchitectureAwareBalancer(t *testing.T) {
	targets := []*middleware.ProxyTarget{
		newTarget("arm1", container.ARM),
		newTarget("x86_1", container.X86),
		newTarget("arm2", container.ARM),
	}

	b := NewArchitectureAwareBalancer(targets)

	assert.Equal(t, 2, b.armRing.Size())
	assert.Equal(t, 1, b.x86Ring.Size())
}

func TestAddTarget(t *testing.T) {
	b := NewArchitectureAwareBalancer([]*middleware.ProxyTarget{})
	b.AddTarget(newTarget("arm1", container.ARM))
	b.AddTarget(newTarget("x86_1", container.X86))

	assert.Equal(t, 1, b.armRing.Size())
	assert.Equal(t, 1, b.x86Ring.Size())
}

func TestRemoveTarget(t *testing.T) {
	targets := []*middleware.ProxyTarget{
		newTarget("arm1", container.ARM),
		newTarget("x86_1", container.X86),
	}
	b := NewArchitectureAwareBalancer(targets)

	assert.Equal(t, 1, b.armRing.Size())
	assert.Equal(t, 1, b.x86Ring.Size())

	assert.True(t, b.RemoveTarget("arm1"))
	assert.False(t, b.RemoveTarget("unknown"))
	assert.Equal(t, 0, b.armRing.Size())
	assert.Equal(t, 1, b.x86Ring.Size())
}

func TestSelectArchitecture(t *testing.T) {
	targets := []*middleware.ProxyTarget{
		newTarget("arm1", container.ARM),
		newTarget("x86_1", container.X86),
	}
	b := NewArchitectureAwareBalancer(targets)

	// Test case 1: Function supports both ARM and x86
	funBoth := &function.Function{Name: "bothArchs", SupportedArchs: []string{container.X86, container.ARM}}
	arch, err := b.selectArchitecture(funBoth)
	assert.NoError(t, err)
	assert.Equal(t, container.ARM, arch)

	// Test case 2: Function supports only ARM
	funArm := &function.Function{Name: "onlyArm", SupportedArchs: []string{container.ARM}}
	arch, err = b.selectArchitecture(funArm)
	assert.NoError(t, err)
	assert.Equal(t, container.ARM, arch)

	// Test case 3: Function supports only x86
	funX86 := &function.Function{Name: "onlyX86", SupportedArchs: []string{container.X86}}
	arch, err = b.selectArchitecture(funX86)
	assert.NoError(t, err)
	assert.Equal(t, container.X86, arch)

	// Test case 4: No available nodes for supported architecture
	b.RemoveTarget("arm1")
	b.RemoveTarget("x86_1")
	_, err = b.selectArchitecture(funBoth)
	assert.Error(t, err)
}

func TestConsistentNodeMapping(t *testing.T) {
	targets := []*middleware.ProxyTarget{
		newTarget("arm1", container.ARM),
		newTarget("x86_1", container.X86),
		newTarget("arm2", container.ARM),
		newTarget("x86_2", container.X86),
	}
	b := NewArchitectureAwareBalancer(targets)

	fun := &function.Function{
		Name:           "testFunc",
		SupportedArchs: []string{container.ARM, container.X86},
	}

	// Add the function to the cache to avoid etcd dependency
	cache.GetCacheInstance().Set(fun.Name, fun, 30*time.Second)
	defer cache.GetCacheInstance().Delete(fun.Name)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/invoke/testFunc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// First call
	firstTarget := b.Next(c)
	assert.NotNil(t, firstTarget)

	// Subsequent calls should return the same target
	for i := 0; i < 10; i++ {
		nextTarget := b.Next(c)
		assert.Equal(t, firstTarget, nextTarget)
	}
}
