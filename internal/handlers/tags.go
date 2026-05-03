package handlers

import (
	"net/http"

	"holo/internal/components"
	db "holo/internal/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, _ := h.queries.ListTags(r.Context())
	spendByTag, _ := h.queries.GetSpendByTag(r.Context(), db.GetSpendByTagParams{
		DateFrom: "",
		DateTo:   "",
	})
	components.TagsSettingsSection(tags, spendByTag).Render(r.Context(), w)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	color := r.FormValue("color")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if color == "" {
		color = "#64748b"
	}
	if _, err := h.queries.CreateTag(r.Context(), db.CreateTagParams{
		ID:    uuid.New().String(),
		Name:  name,
		Color: color,
	}); err != nil {
		http.Error(w, "create tag failed", http.StatusInternalServerError)
		return
	}
	tags, _ := h.queries.ListTags(r.Context())
	spendByTag, _ := h.queries.GetSpendByTag(r.Context(), db.GetSpendByTagParams{
		DateFrom: "",
		DateTo:   "",
	})
	components.TagsSettingsSection(tags, spendByTag).Render(r.Context(), w)
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.queries.DeleteTag(r.Context(), id); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	tags, _ := h.queries.ListTags(r.Context())
	spendByTag, _ := h.queries.GetSpendByTag(r.Context(), db.GetSpendByTagParams{
		DateFrom: "",
		DateTo:   "",
	})
	components.TagsSettingsSection(tags, spendByTag).Render(r.Context(), w)
}

type TagHandler struct {
	queries *db.Queries
}

func NewTagHandler(queries *db.Queries) *TagHandler {
	return &TagHandler{queries: queries}
}

func (h *TagHandler) Add(w http.ResponseWriter, r *http.Request) {
	txnID := chi.URLParam(r, "id")

	tagID := r.FormValue("tag_id")
	tagName := r.FormValue("tag_name")

	switch {
	case tagID != "":
		if err := h.queries.AddTagToTransaction(r.Context(), db.AddTagToTransactionParams{
			TransactionID: txnID,
			TagID:         tagID,
		}); err != nil {
			http.Error(w, "add tag failed", http.StatusInternalServerError)
			return
		}
	case tagName != "":
		// preserve existing color if the tag already exists
		existing, lookupErr := h.queries.GetTagByName(r.Context(), tagName)
		var tag db.Tag
		var err error
		if lookupErr == nil {
			tag = existing
		} else {
			tag, err = h.queries.UpsertTag(r.Context(), db.UpsertTagParams{
				ID:    uuid.New().String(),
				Name:  tagName,
				Color: "#6366f1",
			})
		}
		if err != nil {
			http.Error(w, "upsert tag failed", http.StatusInternalServerError)
			return
		}
		if err := h.queries.AddTagToTransaction(r.Context(), db.AddTagToTransactionParams{
			TransactionID: txnID,
			TagID:         tag.ID,
		}); err != nil {
			http.Error(w, "add tag failed", http.StatusInternalServerError)
			return
		}
	}

	h.renderTagsCell(w, r, txnID)
}

func (h *TagHandler) Remove(w http.ResponseWriter, r *http.Request) {
	txnID := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tag_id")

	if err := h.queries.RemoveTagFromTransaction(r.Context(), db.RemoveTagFromTransactionParams{
		TransactionID: txnID,
		TagID:         tagID,
	}); err != nil {
		http.Error(w, "remove failed", http.StatusInternalServerError)
		return
	}

	h.renderTagsCell(w, r, txnID)
}

func (h *TagHandler) renderTagsCell(w http.ResponseWriter, r *http.Request, txnID string) {
	tags, _ := h.queries.ListTagsForTransaction(r.Context(), txnID)
	allTags, _ := h.queries.ListTags(r.Context())
	components.TxnTagsCell(txnID, tags, allTags).Render(r.Context(), w)
}
