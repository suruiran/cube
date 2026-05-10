package vld

import "context"

type ValidateFunction func(ctx context.Context, val any) (bool, error)

type Validator interface {
	Validate() (bool, error)
}
