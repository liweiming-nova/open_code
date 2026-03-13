package session

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestJSONStoreCRUD(t *testing.T) {
	t.Parallel()

	store, err := NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}

	ctx := context.Background()
	id := "test_session_1"

	s := &Session{
		ID:       id,
		Name:     "test",
		Messages: []adk.Message{schema.UserMessage("hello")},
	}
	if err = store.Save(ctx, s); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != id {
		t.Fatalf("Get().ID = %q, want %q", got.ID, id)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("len(Get().Messages) = %d, want 1", len(got.Messages))
	}

	got, err = store.AppendMessages(ctx, id, schema.AssistantMessage("hi", nil))
	if err != nil {
		t.Fatalf("AppendMessages() error = %v", err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(AppendMessages().Messages) = %d, want 2", len(got.Messages))
	}

	got, err = store.ClearMessages(ctx, id)
	if err != nil {
		t.Fatalf("ClearMessages() error = %v", err)
	}
	if len(got.Messages) != 0 {
		t.Fatalf("len(ClearMessages().Messages) = %d, want 0", len(got.Messages))
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(List()) = %d, want 1", len(list))
	}

	if err = store.Delete(ctx, id); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = store.Get(ctx, id)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Get() after delete error = %v, want ErrSessionNotFound", err)
	}
}

func TestJSONStoreInvalidID(t *testing.T) {
	t.Parallel()

	store, err := NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore() error = %v", err)
	}

	_, err = store.Get(context.Background(), "../bad")
	if err == nil {
		t.Fatal("Get() with invalid id should fail")
	}
}
