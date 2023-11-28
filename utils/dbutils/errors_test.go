package dbutils_test

import (
	"testing"

	"github.com/lib/pq"
	"go-dispatcher2/utils/dbutils"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueViolation(t *testing.T) {
	var err error = &pq.Error{Code: pq.ErrorCode("23505")}

	assert.True(t, dbutils.IsUniqueViolation(err))
	assert.False(t, dbutils.IsUniqueViolation(errors.New("boom")))
}
