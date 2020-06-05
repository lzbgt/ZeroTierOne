/*
 * Copyright (c)2013-2020 ZeroTier, Inc.
 *
 * Use of this software is governed by the Business Source License included
 * in the LICENSE.TXT file in the project's root directory.
 *
 * Change Date: 2024-01-01
 *
 * On the date above, in accordance with the Business Source License, use
 * of this software will be governed by version 2.0 of the Apache License.
 */
/****/

package zerotier

// #include "../../serviceiocore/GoGlue.h"
import "C"

import (
	"encoding/json"
	"runtime"
	"unsafe"
)

type Locator struct {
	Timestamp   int64        `json:"timestamp"`
	Fingerprint *Fingerprint `json:"fingerprint"`
	Endpoints   []Endpoint   `json:"endpoints"`
	String      string       `json:"string"`
	cl          unsafe.Pointer
}

func NewLocator(ts int64, endpoints []Endpoint, signer *Identity) (*Locator, error) {
	if ts <= 0 || len(endpoints) == 0 || signer == nil {
		return nil, ErrInvalidParameter
	}
	eps := make([]C.ZT_Endpoint, 0, len(endpoints))
	for _, e := range endpoints {
		eps = append(eps, e.cep)
	}
	signer.initCIdentityPtr()
	loc := C.ZT_Locator_create(C.int64_t(ts), &eps[0], C.uint(len(eps)), signer.cid)
	if uintptr(loc) == 0 {
		return nil, ErrInvalidParameter
	}

	goloc := new(Locator)
	goloc.cl = unsafe.Pointer(loc)
	return goloc, goloc.init()
}

func NewLocatorFromBytes(lb []byte) (*Locator, error) {
	if len(lb) == 0 {
		return nil, ErrInvalidParameter
	}
	loc := C.ZT_Locator_unmarshal(unsafe.Pointer(&lb[0]), C.uint(len(lb)))
	if uintptr(loc) == 0 {
		return nil, ErrInvalidParameter
	}

	goloc := new(Locator)
	goloc.cl = unsafe.Pointer(loc)
	return goloc, goloc.init()
}

func NewLocatorFromString(s string) (*Locator, error) {
	if len(s) == 0 {
		return nil, ErrInvalidParameter
	}
	sb := []byte(s)
	sb = append(sb, 0)
	loc := C.ZT_Locator_fromString((*C.char)(unsafe.Pointer(&sb[0])))
	if loc == nil {
		return nil, ErrInvalidParameter
	}

	goloc := new(Locator)
	goloc.cl = unsafe.Pointer(loc)
	return goloc, goloc.init()
}

func (loc *Locator) Validate(id *Identity) bool {
	if id == nil {
		return false
	}
	id.initCIdentityPtr()
	return C.ZT_Locator_verify(loc.cl, id.cid) != 0
}

func (loc *Locator) Bytes() []byte {
	var buf [4096]byte
	bl := C.ZT_Locator_marshal(loc.cl, unsafe.Pointer(&buf[0]), 4096)
	if bl <= 0 {
		return nil
	}
	return buf[0:int(bl)]
}

func (loc *Locator) MarshalJSON() ([]byte, error) {
	return json.Marshal(loc)
}

func (loc *Locator) UnmarshalJSON(j []byte) error {
	C.ZT_Locator_delete(loc.cl)
	loc.cl = unsafe.Pointer(nil)

	err := json.Unmarshal(j, loc)
	if err != nil {
		return err
	}

	sb := []byte(loc.String)
	sb = append(sb, 0)
	cl := C.ZT_Locator_fromString((*C.char)(unsafe.Pointer(&sb[0])))
	if cl == nil {
		return ErrInvalidParameter
	}
	loc.cl = cl
	return loc.init()
}

func locatorFinalizer(obj interface{}) {
	if obj != nil {
		C.ZT_Locator_delete(obj.(Locator).cl)
	}
}

func (loc *Locator) init() error {
	loc.Timestamp = int64(C.ZT_Locator_timestamp(loc.cl))
	cfp := C.ZT_Locator_fingerprint(loc.cl)
	if uintptr(unsafe.Pointer(cfp)) == 0 {
		return ErrInternal
	}
	loc.Fingerprint = newFingerprintFromCFingerprint(cfp)
	epc := int(C.ZT_Locator_endpointCount(loc.cl))
	loc.Endpoints = make([]Endpoint, epc)
	for i := 0; i < epc; i++ {
		loc.Endpoints[i].cep = *C.ZT_Locator_endpoint(loc.cl, C.uint(i))
	}
	var buf [4096]byte
	loc.String = C.GoString(C.ZT_Locator_toString(loc.cl, (*C.char)(unsafe.Pointer(&buf[0])), 4096))
	runtime.SetFinalizer(loc, locatorFinalizer)
	return nil
}
