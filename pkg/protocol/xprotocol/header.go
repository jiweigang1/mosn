/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xprotocol

import (
	"reflect"
	"unsafe"
)

// BytesKV key-value pair in byte slice
// key value Map key 其实是String 使用 byte 方式存储，可以减少后续的 String 字节转换处理 
type BytesKV struct {
	Key   []byte
	Value []byte
}

// Header consists of multi key-value pair in byte slice formation. This could reduce the cost of []byte to string for protocol codec.
// 通用的 Header 结构
type Header struct {
	Kvs []BytesKV
	Changed bool // Header 是否发生变化，设置了Header的值就表示为true 标示发生了变化
}

// ~ HeaderMap
// 获取 Header 的值 Value 也是String
func (h *Header) Get(Key string) (Value string, ok bool) {
	//遍历所有的 KV 获取匹配的值
	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		if Key == string(kv.Key) {
			return string(kv.Value), true
		}
	}
	return "", false
}
// 设置值值
func (h *Header) Set(Key string, Value string) {
	//标示Header 发生了变化
	h.Changed = true
	//遍历所有的KV，
	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		//如果是已经存在，追加到值的内容中
		if Key == string(kv.Key) {
			kv.Value = append(kv.Value[:0], Value...)
			return
		}
	}
	// 如果不存在添加新的KV进行存储
	var kv *BytesKV
	h.Kvs, kv = allocKV(h.Kvs)
	kv.Key = append(kv.Key[:0], Key...)
	kv.Value = append(kv.Value[:0], Value...)
}

func (h *Header) Add(Key string, Value string) {
	panic("not supported")
}

func (h *Header) Del(Key string) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		if Key == string(kv.Key) {
			h.Changed = true

			tmp := *kv
			copy(h.Kvs[i:], h.Kvs[i+1:])
			n--
			h.Kvs[n] = tmp
			h.Kvs = h.Kvs[:n]
			return
		}
	}
}

func (h *Header) Range(f func(Key, Value string) bool) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		kv := &h.Kvs[i]
		// false means stop iteration
		if !f(b2s(kv.Key), b2s(kv.Value)) {
			return
		}
	}
}

func (h *Header) Clone() *Header {
	n := len(h.Kvs)

	clone := &Header{
		Kvs: make([]BytesKV, n),
	}

	for i := 0; i < n; i++ {
		src := &h.Kvs[i]
		dst := &clone.Kvs[i]

		dst.Key = append(dst.Key[:0], src.Key...)
		dst.Value = append(dst.Value[:0], src.Value...)
	}

	return clone
}

func (h *Header) ByteSize() (size uint64) {
	for _, kv := range h.Kvs {
		size += uint64(len(kv.Key) + len(kv.Value))
	}
	return
}
//申请新的KV结构
func allocKV(h []BytesKV) ([]BytesKV, *BytesKV) {
	n := len(h)
	// 如果数组容量够用，直接在添加在末尾
	if cap(h) > n {
		h = h[:n+1]
	//扩充数组	
	} else {
		h = append(h, BytesKV{})
	}
	return h, &h[n]
}

// b2s converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
//
// Note it may break if string and/or slice header will change
// in the future go versions.
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// s2b converts string to a byte slice without memory allocation.
//
// Note it may break if string and/or slice header will change
// in the future go versions.
func s2b(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
