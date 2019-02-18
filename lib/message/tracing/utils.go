// Copyright (c) 2019 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package tracing

import (
	"context"

	"github.com/Jeffail/benthos/lib/message"
	"github.com/Jeffail/benthos/lib/types"
	"github.com/opentracing/opentracing-go"
)

// IterateWithSpan iterates all the parts of a message and, for each part,
// creates a new span from an existing span attached to the part and calls a
// func with that span.
func IterateWithSpan(msg types.Message, operationName string, iter func(int, opentracing.Span, types.Part) error) error {
	return msg.Iter(func(i int, p types.Part) error {
		span, _ := opentracing.StartSpanFromContext(message.GetContext(p), operationName)
		err := iter(i, span, p)
		span.Finish()
		return err
	})
}

// GetSpan returns a span attached to a message part. Returns nil if the part
// doesn't have a span attached.
func GetSpan(p types.Part) opentracing.Span {
	return opentracing.SpanFromContext(message.GetContext(p))
}

// InitSpans sets up OpenTracing spans on each message part if one does not
// already exist.
func InitSpans(operationName string, msg types.Message) {
	tracedParts := make([]types.Part, msg.Len())
	msg.Iter(func(i int, p types.Part) error {
		if GetSpan(p) != nil {
			tracedParts[i] = p
		}
		span := opentracing.StartSpan(operationName)
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		tracedParts[i] = message.WithContext(ctx, p)
		return nil
	})
	msg.SetAll(tracedParts)
}

// InitSpansFromParent sets up OpenTracing spans that are children of a parent
// span on each message part.
func InitSpansFromParent(parent opentracing.SpanContext, operationName string, msg types.Message) {
	tracedParts := make([]types.Part, msg.Len())
	msg.Iter(func(i int, p types.Part) error {
		span := opentracing.StartSpan(operationName, opentracing.ChildOf(parent))
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		tracedParts[i] = message.WithContext(ctx, p)
		return nil
	})
	msg.SetAll(tracedParts)
}

// FinishSpans calls Finish on all message parts containing a span.
func FinishSpans(msg types.Message) {
	msg.Iter(func(i int, p types.Part) error {
		span := GetSpan(p)
		if span == nil {
			return nil
		}
		span.Finish()
		return nil
	})
}
