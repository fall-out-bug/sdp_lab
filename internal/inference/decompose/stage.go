package decompose

import "context"

// Stage is the unit of work inside a Pipeline. In and Out are the input and
// output types of this stage; they must match adjacent stages in the chain.
type Stage[In, Out any] interface {
	Name() string
	Run(ctx context.Context, in In) (Out, StageTrace, error)
}

// funcStage wraps a plain function as a Stage.
type funcStage[In, Out any] struct {
	name string
	fn   func(ctx context.Context, in In) (Out, StageTrace, error)
}

func (s *funcStage[In, Out]) Name() string { return s.name }

func (s *funcStage[In, Out]) Run(ctx context.Context, in In) (Out, StageTrace, error) {
	return s.fn(ctx, in)
}

// NewStage constructs a Stage from a function. The returned Stage has the
// given name and delegates Run to fn.
func NewStage[In, Out any](name string, fn func(ctx context.Context, in In) (Out, StageTrace, error)) Stage[In, Out] {
	return &funcStage[In, Out]{name: name, fn: fn}
}

// anyStage is a type-erased stage stored inside Pipeline. The pipeline calls
// runAny with an untyped input and receives an untyped output.
type anyStage interface {
	stageName() string
	runAny(ctx context.Context, in any) (any, StageTrace, error)
}

// typedAnyStage adapts Stage[In,Out] to anyStage.
type typedAnyStage[In, Out any] struct {
	inner Stage[In, Out]
}

func (a *typedAnyStage[In, Out]) stageName() string { return a.inner.Name() }

func (a *typedAnyStage[In, Out]) runAny(ctx context.Context, in any) (any, StageTrace, error) {
	typed, ok := in.(In)
	if !ok {
		var zero In
		panic("decompose: stage " + a.inner.Name() + " expected input type " +
			typeName(zero) + " but got " + typeName(in))
	}
	out, trace, err := a.inner.Run(ctx, typed)
	return out, trace, err
}

// wrapAny wraps a Stage[In,Out] into anyStage.
func wrapAny[In, Out any](s Stage[In, Out]) anyStage {
	return &typedAnyStage[In, Out]{inner: s}
}
