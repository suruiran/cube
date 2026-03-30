package cube

import "context"

type ISecretKeeper interface {
	Encrypt(ctx context.Context, raw string) (string, string, error)
	Decrypt(ctx context.Context, enc string, salt string) (string, error)
}
