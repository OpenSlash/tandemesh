package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func (s *Server) attachmentDir() string {
	if dir := os.Getenv("ORCA_ATTACHMENT_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(filepath.Dir(s.cfg.DBPath), "attachments")
}

// downloadTaskAttachment streams an attachment through the control plane and
// keeps a private local cache.
func (s *Server) downloadTaskAttachment(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	attachment, err := s.store.GetTaskAttachment(r.Context(), chi.URLParam(r, "attachmentId"))
	if err != nil || attachment == nil || attachment.TaskID != task.ID {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "attachment not found")
		return
	}
	sum := sha256.Sum256([]byte(attachment.Provider + "|" + attachment.ExternalID + "|" + attachment.URL))
	cachePath := filepath.Join(s.attachmentDir(), hex.EncodeToString(sum[:]))
	if _, err := os.Stat(cachePath); err != nil {
		if err := s.fetchAttachment(r, task, attachment, cachePath); err != nil {
			writeErr(w, http.StatusBadGateway, "E_ATTACHMENT", err.Error())
			return
		}
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(attachment.Name)+`"`)
	http.ServeFile(w, r, cachePath)
}

func (s *Server) fetchAttachment(r *http.Request, task *domain.Task, attachment *domain.TaskAttachment, cachePath string) error {
	ad, ok := s.integrations.Get(task.Provider)
	if !ok {
		return integration.ErrUnsupported
	}
	reader, ok := ad.Work.(integration.AttachmentReader)
	if !ok {
		return integration.ErrUnsupported
	}
	binding, _ := s.store.FindBinding(r.Context(), task.Provider, task.Instance)
	connectionID := ""
	accountID := ""
	if binding != nil {
		connectionID = binding.ConnectionID
		accountID = binding.AccountID
	}
	cred := s.providerCredential(r.Context(), task.Provider, connectionID)
	cred = s.accountCredential(r.Context(), task.Provider, accountID, cred)
	body, _, err := reader.OpenAttachment(r.Context(), cred, integration.Attachment{
		ID: attachment.ExternalID, Name: attachment.Name, URL: attachment.URL, Mime: attachment.Mime,
	})
	if err != nil {
		return err
	}
	defer body.Close()
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(cachePath), ".attachment-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), cachePath)
}
