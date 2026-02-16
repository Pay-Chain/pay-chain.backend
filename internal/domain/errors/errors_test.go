package errors

import (
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Constructors(t *testing.T) {
	err := NewAppError(http.StatusBadRequest, CodeBadRequest, "bad", ErrBadRequest)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, CodeBadRequest, err.Code)
	assert.Equal(t, "bad", err.Message)
	assert.Equal(t, ErrBadRequest.Error(), err.Error())

	notFound := NotFound("missing")
	assert.Equal(t, http.StatusNotFound, notFound.Status)
	assert.Equal(t, CodeNotFound, notFound.Code)

	conflict := Conflict("exists")
	assert.Equal(t, http.StatusConflict, conflict.Status)
	assert.Equal(t, CodeConflict, conflict.Code)

	internal := InternalError(stderrors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, internal.Status)
	assert.Equal(t, CodeInternalError, internal.Code)

	custom := NewError("custom", ErrForbidden)
	assert.Equal(t, ErrForbidden.Error(), custom.Error())

	badReq := BadRequest("bad request")
	assert.Equal(t, http.StatusBadRequest, badReq.Status)
	assert.Equal(t, CodeInvalidInput, badReq.Code)

	unauth := Unauthorized("unauthorized")
	assert.Equal(t, http.StatusUnauthorized, unauth.Status)
	assert.Equal(t, CodeUnauthorized, unauth.Code)

	forbidden := Forbidden("forbidden")
	assert.Equal(t, http.StatusForbidden, forbidden.Status)
	assert.Equal(t, CodeForbidden, forbidden.Code)

	internalMsg := InternalServerError("boom")
	assert.Equal(t, http.StatusInternalServerError, internalMsg.Status)
	assert.Equal(t, "boom", internalMsg.Message)
	assert.Equal(t, "boom", internalMsg.Error())
}
