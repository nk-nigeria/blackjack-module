package presenter

import "github.com/heroiclabs/nakama-common/runtime"

var (
	ErrInternalError  = runtime.NewError("internal server error", 13) // INTERNAL
	ErrMarshal        = runtime.NewError("cannot marshal type", 13)   // INTERNAL
	ErrNoInputAllowed = runtime.NewError("no input allowed", 3)       // INVALID_ARGUMENT
	ErrNoUserIdFound  = runtime.NewError("no user ID in context", 3)  // INVALID_ARGUMENT
	ErrUnmarshal      = runtime.NewError("cannot unmarshal type", 13) // INTERNAL

	ErrGameFinish = runtime.NewError("game finish", 100) // INTERNAL
)
