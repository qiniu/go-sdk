package http_client

import "context"

type noSignatureContextKey struct{}

func WithoutSignature(ctx context.Context) context.Context {
	return context.WithValue(ctx, noSignatureContextKey{}, struct{}{})
}

func isSignatureDisabled(ctx context.Context) bool {
	_, ok := ctx.Value(noSignatureContextKey{}).(struct{})
	return ok
}
