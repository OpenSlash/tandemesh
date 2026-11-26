package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

// registryIDPattern limits registry ids and versions to safe path segments;
// the host uses them to build directories under its ACP home.
var registryIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ACPInstallPayload is the payload of acp-install and acp-uninstall jobs.
type ACPInstallPayload struct {
	RegistryID string `json:"registryId"`
	// Version selects one install to remove; empty removes every version.
	Version string `json:"version,omitempty"`
}

// MachineJobResult is what a host reports when a job finishes. Runtime
// fields carry the host's detection after the job, so the UI updates without
// waiting for the next heartbeat.
type MachineJobResult struct {
	Status         string                 `json:"status"`
	Error          string                 `json:"error,omitempty"`
	Result         json.RawMessage        `json:"result,omitempty"`
	Runtimes       map[string]string      `json:"runtimes,omitempty"`
	RuntimeDetails []domain.RuntimeStatus `json:"runtimeDetails,omitempty"`
	Agents         []string               `json:"agents,omitempty"`
}

func (s *Server) requestRuntimeCheck(w http.ResponseWriter, r *http.Request) {
	s.queueMachineJob(w, r, domain.MachineJobRuntimeCheck, struct{}{})
}

func (s *Server) requestACPInstall(w http.ResponseWriter, r *http.Request) {
	var req ACPInstallPayload
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if !registryIDPattern.MatchString(req.RegistryID) {
		writeErr(w, http.StatusBadRequest, "E_REGISTRY_ID", "registryId is required")
		return
	}
	s.queueMachineJob(w, r, domain.MachineJobACPInstall, ACPInstallPayload{RegistryID: req.RegistryID})
}

func (s *Server) requestACPUninstall(w http.ResponseWriter, r *http.Request) {
	req := ACPInstallPayload{RegistryID: r.URL.Query().Get("registryId"), Version: r.URL.Query().Get("version")}
	if !registryIDPattern.MatchString(req.RegistryID) {
		writeErr(w, http.StatusBadRequest, "E_REGISTRY_ID", "registryId is required")
		return
	}
	if req.Version != "" && !registryIDPattern.MatchString(req.Version) {
		writeErr(w, http.StatusBadRequest, "E_VERSION", "invalid version")
		return
	}
	s.queueMachineJob(w, r, domain.MachineJobACPUninstall, req)
}

// queueMachineJob queues work for an online host. Offline hosts are rejected
// so the UI does not wait on a job nobody will pick up.
func (s *Server) queueMachineJob(w http.ResponseWriter, r *http.Request, kind string, payload any) {
	machine, err := s.store.GetMachine(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if machine == nil {
		writeErr(w, http.StatusNotFound, "E_MACHINE", "machine not found")
		return
	}
	if machine.Status != "online" {
		writeErr(w, http.StatusConflict, "E_MACHINE_OFFLINE", "machine is not online")
		return
	}
	job, err := s.store.CreateMachineJob(r.Context(), machine.ID, kind, payload)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) getMachineJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.store.GetMachineJob(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if job == nil {
		writeErr(w, http.StatusNotFound, "E_JOB", "machine job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) agentdMachineJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ClaimMachineJobs(r.Context(), chi.URLParam(r, "machineId"), 5)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": jobs})
}

func (s *Server) agentdMachineJobResult(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineId")
	var req MachineJobResult
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Status != "done" && req.Status != "failed" {
		writeErr(w, http.StatusBadRequest, "E_STATUS", "status must be done or failed")
		return
	}
	err := s.store.FinishMachineJob(r.Context(), chi.URLParam(r, "id"), machineID, req.Status, req.Result, req.Error)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "E_JOB", "no running job with this id for this machine")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if req.RuntimeDetails != nil {
		if err := s.store.SetMachineRuntimes(r.Context(), machineID, req.Runtimes, req.RuntimeDetails, req.Agents); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

// deleteMachine removes an offline host. Online hosts would re-register on
// their next heartbeat, and hosts with agents or connections would strand
// them, so both are refused.
func (s *Server) deleteMachine(w http.ResponseWriter, r *http.Request) {
	machine, err := s.store.GetMachine(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if machine == nil {
		writeErr(w, http.StatusNotFound, "E_MACHINE", "machine not found")
		return
	}
	if machine.Status == "online" {
		writeErr(w, http.StatusConflict, "E_MACHINE_ONLINE", "only offline machines can be deleted")
		return
	}
	err = s.store.DeleteMachine(r.Context(), machine.ID)
	if errors.Is(err, store.ErrMachineInUse) {
		writeErr(w, http.StatusConflict, "E_MACHINE_IN_USE", "move this machine's agents to another host first")
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
