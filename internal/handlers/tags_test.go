package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	db "holo/internal/db/generated"
	"holo/internal/testutil"

	"github.com/google/uuid"
)

func newTestTagHandler(t *testing.T) *TagHandler {
	t.Helper()
	q := testutil.NewDB(t)
	return &TagHandler{queries: q}
}

func TestTagsPage_Returns200_NoTags(t *testing.T) {
	h := newTestTagHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	rr := httptest.NewRecorder()

	h.Page(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestTagsPage_Returns200_WithData(t *testing.T) {
	q := testutil.NewDB(t)
	ctx := context.Background()

	// Insert institution and account
	instID := testutil.InsertInstitution(t, q)
	acctID := testutil.InsertAccount(t, q, instID, "checking")

	// Insert a transaction
	testutil.InsertTransaction(t, q, "txn-tag-1", acctID, "plaid-tag-1", "Coffee Shop", 12.50, "", "")

	// Create a tag
	tag, err := q.CreateTag(ctx, db.CreateTagParams{
		ID:    uuid.New().String(),
		Name:  "Coffee",
		Color: "#a78bfa",
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	// Tag the transaction
	if err := q.AddTagToTransaction(ctx, db.AddTagToTransactionParams{
		TransactionID: "txn-tag-1",
		TagID:         tag.ID,
	}); err != nil {
		t.Fatalf("add tag to transaction: %v", err)
	}

	h := &TagHandler{queries: q}
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	rr := httptest.NewRecorder()

	h.Page(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Coffee") {
		t.Errorf("expected body to contain tag name 'Coffee', got:\n%s", rr.Body.String())
	}
}

func TestTagsPage_Returns200_WithDateFilter(t *testing.T) {
	h := newTestTagHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/tags?date_from=2026-01-01&date_to=2026-01-31", nil)
	rr := httptest.NewRecorder()

	h.Page(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
