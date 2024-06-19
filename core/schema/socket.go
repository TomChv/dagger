package schema

import (
	"context"

	"github.com/dagger/dagger/core"
	"github.com/dagger/dagger/dagql"
	"github.com/dagger/dagger/engine/slog"
)

type socketSchema struct {
	srv *dagql.Server
}

var _ SchemaResolvers = &socketSchema{}

func (s *socketSchema) Install() {
	dagql.Fields[*core.Query]{
		dagql.Func("socket", s.socket).
			Doc("Loads a socket by its ID.").
			Deprecated("Use `loadSocketFromID` instead."),
	}.Install(s.srv)

	dagql.Fields[*core.Socket]{}.Install(s.srv)
}

type socketArgs struct {
	ID core.SocketID
}

func (s *socketSchema) socket(ctx context.Context, parent *core.Query, args socketArgs) (dagql.Instance[*core.Socket], error) {
	slog.Warn("socket is deprecated, use loadSocketFromID instead")

	return args.ID.Load(ctx, s.srv)
}
