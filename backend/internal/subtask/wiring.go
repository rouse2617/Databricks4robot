package subtask

import "context"

type BatchCreatorFn func(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int, owner string) (string, error)

func (f BatchCreatorFn) CreateBatch(ctx context.Context, tmpl, name string, ids []string, target string, ver int, owner string) (string, error) {
	return f(ctx, tmpl, name, ids, target, ver, owner)
}
