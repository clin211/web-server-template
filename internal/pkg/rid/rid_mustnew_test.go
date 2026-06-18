package rid_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/clin211/gin-enterprise-template/internal/pkg/rid"
)

func TestResourceIDMustNew(t *testing.T) {
	t.Parallel()

	first := rid.UserID.MustNew()
	second := rid.UserID.MustNew()

	fmt.Println(first)
	fmt.Println(second)
	assert.True(t, strings.HasPrefix(first, rid.UserID.String()+"-"))
	assert.True(t, strings.HasPrefix(second, rid.UserID.String()+"-"))
	assert.Len(t, strings.SplitN(first, "-", 2), 2)
	assert.Len(t, strings.SplitN(second, "-", 2), 2)
	assert.NotEqual(t, first, second)
}
