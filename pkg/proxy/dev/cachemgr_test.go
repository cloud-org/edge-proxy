package dev

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestSlice(t *testing.T) {
	// io.ReadAll func 会用到 b = append(b, 0)[:len(b)]
	a := make([]byte, 0, 10)
	t.Log(len(a), cap(a))
	a = append(a, 0)[:10]
	t.Log(len(a), cap(a))
	a = append(a, 0)[:10]
	t.Log(len(a), cap(a))
	//    cachemgr_test.go:13: 0 10
	//    cachemgr_test.go:15: 10 10
	//    cachemgr_test.go:17: 10 24
	b := make([]byte, 0, 0)
	b = append(b, 1)
	t.Log(len(b), cap(b), b[:6])
	b = b[:6]
	// 通过 [:6] 是可以取到 cap 长度内的切片，然后 b = b[:6] 会将 b.len 进行设置，请注意 cap 还是不变
	t.Log(b, len(b), cap(b))
	//    cachemgr_test.go:23: 1 8 [1 0 0 0 0 0]
	//    cachemgr_test.go:26: [1 0 0 0 0 0] 6 8
}

func TestReadFrom(t *testing.T) {
	a := strings.NewReader("hello world")
	//p := bytes.NewBuffer(make([]byte, 0, 20)) // cap 552
	p := bytes.NewBuffer(nil) // cap 1536
	n, err := p.ReadFrom(a)
	if err != nil {
		t.Error(err)
		return
	}
	data := p.Bytes()
	t.Logf("n is %v, p.len: %v, data: %v, data.cap: %v\n", n, len(data), string(data), cap(data))

	return
}

func TestIoReadAll(t *testing.T) {
	a := strings.NewReader("hello world")
	data, err := io.ReadAll(a)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("p.len: %v, data: %v, data.cap: %v\n", len(data), string(data), cap(data)) // data.cap: 512
}
