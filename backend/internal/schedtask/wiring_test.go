package schedtask

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/schedtask/source"
)

func TestEnvSecretResolver_PlainValue(t *testing.T) {
	t.Setenv("MY_SECRET", "hunter2")
	got, err := (EnvSecretResolver{}).Resolve(context.Background(), "MY_SECRET")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hunter2" {
		t.Fatalf("got %q, want hunter2", got)
	}
}

func TestEnvSecretResolver_LegacyGraceJSONBlobHasAuthPassword(t *testing.T) {
	t.Setenv("GRACE_LEGACY", `{"AUTH_USERNAME":"u","AUTH_PASSWORD":"pw!"}`)
	got, err := (EnvSecretResolver{}).Resolve(context.Background(), "GRACE_LEGACY")
	if err != nil {
		t.Fatal(err)
	}
	if got != "pw!" {
		t.Fatalf("got %q, want pw! (single-blob AUTH_PASSWORD extract)", got)
	}
}

func TestEnvSecretResolver_SubKeyRef(t *testing.T) {
	t.Setenv("MULTI", `{"A":"1","B":"two"}`)
	got, err := (EnvSecretResolver{}).Resolve(context.Background(), "MULTI#B")
	if err != nil {
		t.Fatal(err)
	}
	if got != "two" {
		t.Fatalf("got %q, want two", got)
	}
}

func TestEnvSecretResolver_UnquotedJSONKeys(t *testing.T) {
	t.Setenv("MULTI2", `{ AUTH_USERNAME: "u", AUTH_PASSWORD: "pw" }`)
	got, err := (EnvSecretResolver{}).Resolve(context.Background(), "MULTI2#AUTH_PASSWORD")
	if err != nil {
		t.Fatal(err)
	}
	if got != "pw" {
		t.Fatalf("got %q, want pw (unquoted-key JSON)", got)
	}
}

func TestEnvSecretResolver_EmptyRefOrEnv(t *testing.T) {
	if _, err := (EnvSecretResolver{}).Resolve(context.Background(), ""); err == nil {
		t.Fatal("expected error on empty ref")
	}
	t.Setenv("UNSET_ENV_NAME_A", "")
	if _, err := (EnvSecretResolver{}).Resolve(context.Background(), "UNSET_ENV_NAME_A"); err == nil {
		t.Fatal("expected error on empty env value")
	}
}

func TestSourceFactory_RestOnlyV1(t *testing.T) {
	f := NewDefaultSourceFactory(EnvSecretResolver{})
	if _, err := f.Build("rest", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("rest build failed: %v", err)
	}
	if _, err := f.Build("graphql", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error for unsupported source_type")
	}
}

func TestBatchCreatorFn_Adapts(t *testing.T) {
	var (
		gotTmpl, gotOwner string
		gotIDs            []string
	)
	fn := BatchCreatorFn(func(_ context.Context, tmpl, _ string, ids []string, _ string, _ int, owner string) (string, error) {
		gotTmpl = tmpl
		gotOwner = owner
		gotIDs = ids
		return "batch-adapt-1", nil
	})
	id, err := fn.CreateBatch(context.Background(), "tpl", "n", []string{"a"}, "target", 3, "scheduled-task:r1")
	if err != nil || id != "batch-adapt-1" {
		t.Fatalf("adapt failed: id=%q err=%v", id, err)
	}
	if gotTmpl != "tpl" || gotOwner != "scheduled-task:r1" || len(gotIDs) != 1 {
		t.Fatalf("passthrough wrong: tmpl=%q owner=%q ids=%v", gotTmpl, gotOwner, gotIDs)
	}
}

// silence unused import when tests are trimmed
var _ = source.AssetSource(nil)
