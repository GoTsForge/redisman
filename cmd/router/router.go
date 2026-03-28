package router

import (
	"github.com/gotsforge/redisman/cmd/handler"
	"github.com/gotsforge/redisman/cmd/parser"
	"github.com/gotsforge/redisman/cmd/protocol"
	"github.com/gotsforge/redisman/cmd/server"
)

type Router struct {
	handlers map[string]handler.Handler
}

func NewRouter() Router {
	return Router{
		handlers: map[string]handler.Handler{
			"PING":   handler.HandlePing,
			"SET":    handler.HandleSet,
			"GET":    handler.HandleGet,
			"LPUSH":  handler.HandleLPush,
			"RPUSH":  handler.HandleRPush,
			"LRANGE": handler.HandleLRange,
			"LLEN":   handler.HandleLLen,
			"LPOP":   handler.HandleLPop,
			"BLPOP":  handler.HandleBLPop,
			"TYPE":   handler.HandleType,
			"XADD":   handler.HandleXAdd,
		},
	}
}

func (r Router) Route(s *server.Server, cmd parser.Command) protocol.Response {
	handler, ok := r.handlers[cmd.Name]
	if !ok {
		return protocol.NewErrorResponse("unknown command '" + cmd.Name + "'")
	}

	return handler(s, cmd)
}
