package usecases

import (
	"errors"
	"testing"
)

type rpcDataErrorStub struct {
	msg  string
	data interface{}
}

func (e rpcDataErrorStub) Error() string          { return e.msg }
func (e rpcDataErrorStub) ErrorData() interface{} { return e.data }

func TestDecodeRevertDataFromError_DataErrorString(t *testing.T) {
	err := rpcDataErrorStub{
		msg:  "execution reverted",
		data: "0x4e487b710000000000000000000000000000000000000000000000000000000000000011",
	}
	decoded, ok := decodeRevertDataFromError(err)
	if !ok {
		t.Fatalf("expected decode success")
	}
	if decoded.Name != "Panic" {
		t.Fatalf("expected Panic name, got: %s", decoded.Name)
	}
	if decoded.Message != "panic code: 17" {
		t.Fatalf("unexpected decoded message: %s", decoded.Message)
	}
}

func TestDecodeRevertDataFromError_StringFallback(t *testing.T) {
	err := errors.New("execution reverted: 0xdeadbeef01")
	decoded, ok := decodeRevertDataFromError(err)
	if !ok {
		t.Fatalf("expected decode success from string fallback")
	}
	if decoded.Selector != "0xdeadbeef" {
		t.Fatalf("unexpected selector: %s", decoded.Selector)
	}
}

func TestDecodeRevertDataFromError_NoData(t *testing.T) {
	_, ok := decodeRevertDataFromError(errors.New("execution reverted"))
	if ok {
		t.Fatalf("expected decode failure when no revert data is present")
	}
}
