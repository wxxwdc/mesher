package version

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVer(t *testing.T) {
	versionset := Ver()
	assert.NotNil(t, versionset)
}
