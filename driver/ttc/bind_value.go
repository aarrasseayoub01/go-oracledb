/*
** Copyright (c) 2026 Oracle and/or its affiliates.
**
** The Universal Permissive License (UPL), Version 1.0
**
** Subject to the condition set forth below, permission is hereby granted to any
** person obtaining a copy of this software, associated documentation and data
** (collectively the "Software"), free of charge and under any and all copyright
** rights in the Software, and any and all patent rights owned or freely
** licensable by each licensor hereunder covering either (i) the unmodified
** Software as contributed to or provided by such licensor, or (ii) the Larger
** Works (as defined below), to deal in both
**
** (a) the Software, and
** (b) any piece of software and/or hardware listed in the lrgrwrks.txt file if
** one is included with the Software (each a "Larger Work" to which the Software
** is contributed to or provided by such licensors), and
**
** without restriction, including without limitation the rights to copy, create
** derivative works of, display, perform, and distribute the Software and make,
** use, sell, offer for sale, import, export, have made, and have sold the
** Software and the Larger Work(s), and to sublicense the foregoing rights on
** either these or other terms.
**
** This license is subject to the following condition:
**
** The above copyright notice and either this complete permission notice or at
** a minimum a reference to the UPL must be included in all copies or
** substantial portions of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
** IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
** FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
** AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
** LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
** OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
 */

package ttc

import (
	"context"

	"github.com/oracle/go-driver/driver/common"
)

// bindWireFunc writes one encoded bind value into an outgoing TTIRXD message.
// The default implementation uses the ordinary TTC CLR representation.
type bindWireFunc func(context.Context, common.Marshaller, common.MessageType, int, common.B1Array) error

// bindValue keeps an encoded bind payload together with the wire representation
// selected for its Go type. A nil wire function uses ordinary CLR framing.
type bindValue struct {
	payload common.B1Array
	wire    bindWireFunc
}

func newCLRBindValue(payload common.B1Array) bindValue {
	return bindValue{payload: payload, wire: marshalCLRBind}
}

// marshal writes v with its selected wire representation. A value without a
// registered representation falls back to ordinary TTC CLR framing.
func (v bindValue) marshal(ctx context.Context, mar common.Marshaller, msgType common.MessageType, index int) error {
	if v.wire == nil {
		return marshalCLRBind(ctx, mar, msgType, index, v.payload)
	}
	return v.wire(ctx, mar, msgType, index, v.payload)
}

// marshalCLRBind writes payload as an ordinary TTC CLR bind. A nil payload is
// written as TTC's null length indicator.
func marshalCLRBind(ctx context.Context, mar common.Marshaller, msgType common.MessageType, index int, payload common.B1Array) error {
	if payload == nil {
		if err := mar.MarshalUB1(ctx, common.UB1(0)); err != nil {
			common.Odl.Error("tTIrxd.MarshalTo: failed to write null length indicator",
				"error", err, "stage", "null-indicator", "index", index)
			return common.NewOracleError(common.FailMarshal, err, TTCMsgTypeDescription[msgType])
		}
		return nil
	}

	if err := mar.MarshalCLR(ctx, payload, 0, len(payload)); err != nil {
		common.Odl.Error("tTIrxd.MarshalTo: failed to write CLR",
			"error", err, "stage", "clr", "index", index)
		return common.NewOracleError(common.FailMarshal, err, TTCMsgTypeDescription[msgType])
	}
	return nil
}
