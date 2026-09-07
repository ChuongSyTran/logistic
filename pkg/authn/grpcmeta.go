package authn

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	MetaUserID = "x-user-id"
	MetaRole   = "x-user-role"
	MetaEmail  = "x-user-email"
)

type Identity struct {
	UserID uuid.UUID
	Role   string
	Email  string
}

func (i Identity) IsZero() bool { return i.UserID == uuid.Nil }

func (i Identity) HasRole(roles ...string) bool {
	for _, r := range roles {
		if i.Role == r {
			return true
		}
	}
	return false
}

type identityCtxKey struct{}

var ErrNoIdentity = errors.New("authn: request không mang danh tính")

func Inject(ctx context.Context, id Identity) context.Context {
	if id.IsZero() {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx,
		MetaUserID, id.UserID.String(),
		MetaRole, id.Role,
		MetaEmail, id.Email,
	)
}

func WithIdentity(ctx context.Context, id Identity) context.Context {
	if id.IsZero() {
		return ctx
	}
	return context.WithValue(ctx, identityCtxKey{}, id)
}

func OutgoingIdentityInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if id, ok := FromContext(ctx); ok {
			ctx = Inject(ctx, id)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityCtxKey{}).(Identity)
	return id, ok
}

func MustUserID(ctx context.Context) (uuid.UUID, error) {
	id, ok := FromContext(ctx)
	if !ok || id.IsZero() {
		return uuid.Nil, status.Error(codes.Unauthenticated, "request không mang danh tính")
	}
	return id.UserID, nil
}

func RequireRole(ctx context.Context, roles ...string) error {
	id, ok := FromContext(ctx)
	if !ok || id.IsZero() {
		return status.Error(codes.Unauthenticated, "request không mang danh tính")
	}
	if !id.HasRole(roles...) {
		return status.Error(codes.PermissionDenied, "không đủ quyền")
	}
	return nil
}

func IdentityUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(withIdentityFromMetadata(ctx), req)
	}
}

func IdentityStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &identityStream{ServerStream: ss, ctx: withIdentityFromMetadata(ss.Context())})
	}
}

type identityStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *identityStream) Context() context.Context { return s.ctx }

func withIdentityFromMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	raw := first(md, MetaUserID)
	if raw == "" {
		return ctx
	}
	userID, err := uuid.Parse(raw)
	if err != nil {
		return ctx
	}

	return context.WithValue(ctx, identityCtxKey{}, Identity{
		UserID: userID,
		Role:   first(md, MetaRole),
		Email:  first(md, MetaEmail),
	})
}

func first(md metadata.MD, key string) string {
	if v := md.Get(key); len(v) > 0 {
		return v[0]
	}
	return ""
}