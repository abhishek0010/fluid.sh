package ansible

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	serverError "github.com/aspectrr/fluid.sh/fluid-remote/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/fluid-remote/internal/json"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/store"
)

// PlaybookHandler provides HTTP handlers for playbook management.
type PlaybookHandler struct {
	svc *PlaybookService
}

// NewPlaybookHandler creates a new PlaybookHandler.
func NewPlaybookHandler(svc *PlaybookService) *PlaybookHandler {
	return &PlaybookHandler{svc: svc}
}

// --- Request/Response DTOs ---

type createPlaybookRequest struct {
	Name   string `json:"name"`
	Hosts  string `json:"hosts"`
	Become bool   `json:"become"`
}

type createPlaybookResponse struct {
	Playbook *store.Playbook `json:"playbook"`
}

type getPlaybookResponse struct {
	Playbook *store.Playbook       `json:"playbook"`
	Tasks    []*store.PlaybookTask `json:"tasks"`
}

type listPlaybooksResponse struct {
	Playbooks []*store.Playbook `json:"playbooks"`
	Total     int               `json:"total"`
}

type addTaskRequest struct {
	Name   string         `json:"name"`
	Module string         `json:"module"`
	Params map[string]any `json:"params" swaggertype:"object" `
}

type addTaskResponse struct {
	Task *store.PlaybookTask `json:"task"`
}

type updateTaskRequest struct {
	Name   *string        `json:"name,omitempty"`
	Module *string        `json:"module,omitempty"`
	Params map[string]any `json:"params,omitempty" swaggertype:"object"`
}

type updateTaskResponse struct {
	Task *store.PlaybookTask `json:"task"`
}

type reorderTasksRequest struct {
	TaskIDs []string `json:"task_ids"`
}

type exportPlaybookResponse struct {
	YAML string `json:"yaml"`
}

// --- Handlers ---

// HandleCreatePlaybook creates a new playbook.
// @Summary Create playbook
// @Description Creates a new Ansible playbook
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param request body createPlaybookRequest true "Playbook creation parameters"
// @Success 201 {object} createPlaybookResponse
// @Failure 400 {object} serverError.ErrorResponse
// @Failure 409 {object} serverError.ErrorResponse
// @Id createPlaybook
// @Router /v1/ansible/playbooks [post]
func (h *PlaybookHandler) HandleCreatePlaybook(w http.ResponseWriter, r *http.Request) {
	var req createPlaybookRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name == "" {
		serverError.RespondError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	pb, err := h.svc.CreatePlaybook(r.Context(), CreatePlaybookRequest(req))
	if err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			serverError.RespondError(w, http.StatusConflict, errors.New("playbook with this name already exists"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, createPlaybookResponse{Playbook: pb})
}

// HandleGetPlaybook retrieves a playbook by name.
// @Summary Get playbook
// @Description Gets a playbook and its tasks by name
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param playbook_name path string true "Playbook name"
// @Success 200 {object} getPlaybookResponse
// @Failure 404 {object} serverError.ErrorResponse
// @Id getPlaybook
// @Router /v1/ansible/playbooks/{playbook_name} [get]
func (h *PlaybookHandler) HandleGetPlaybook(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "playbook_name")

	result, err := h.svc.GetPlaybookWithTasksByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, errors.New("playbook not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, getPlaybookResponse{
		Playbook: result.Playbook,
		Tasks:    result.Tasks,
	})
}

// HandleListPlaybooks lists all playbooks.
// @Summary List playbooks
// @Description Lists all Ansible playbooks
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Success 200 {object} listPlaybooksResponse
// @Id listPlaybooks
// @Router /v1/ansible/playbooks [get]
func (h *PlaybookHandler) HandleListPlaybooks(w http.ResponseWriter, r *http.Request) {
	playbooks, err := h.svc.ListPlaybooks(r.Context(), nil)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, listPlaybooksResponse{
		Playbooks: playbooks,
		Total:     len(playbooks),
	})
}

// HandleDeletePlaybook deletes a playbook.
// @Summary Delete playbook
// @Description Deletes a playbook and all its tasks
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param playbook_name path string true "Playbook name"
// @Success 204
// @Failure 404 {object} serverError.ErrorResponse
// @Id deletePlaybook
// @Router /v1/ansible/playbooks/{playbook_name} [delete]
func (h *PlaybookHandler) HandleDeletePlaybook(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "playbook_name")

	pb, err := h.svc.GetPlaybookByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, errors.New("playbook not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	if err := h.svc.DeletePlaybook(r.Context(), pb.ID); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleAddTask adds a task to a playbook.
// @Summary Add task to playbook
// @Description Adds a new task to an existing playbook
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param playbook_name path string true "Playbook name"
// @Param request body addTaskRequest true "Task parameters"
// @Success 201 {object} addTaskResponse
// @Failure 400 {object} serverError.ErrorResponse
// @Failure 404 {object} serverError.ErrorResponse
// @Id addPlaybookTask
// @Router /v1/ansible/playbooks/{playbook_name}/tasks [post]
func (h *PlaybookHandler) HandleAddTask(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "playbook_name")

	pb, err := h.svc.GetPlaybookByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, errors.New("playbook not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	var req addTaskRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name == "" {
		serverError.RespondError(w, http.StatusBadRequest, errors.New("task name is required"))
		return
	}
	if req.Module == "" {
		serverError.RespondError(w, http.StatusBadRequest, errors.New("module is required"))
		return
	}

	task, err := h.svc.AddTask(r.Context(), pb.ID, AddTaskRequest(req))
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, addTaskResponse{Task: task})
}

// HandleUpdateTask updates a task.
// @Summary Update task
// @Description Updates an existing task in a playbook
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param playbook_name path string true "Playbook name"
// @Param task_id path string true "Task ID"
// @Param request body updateTaskRequest true "Task update parameters"
// @Success 200 {object} updateTaskResponse
// @Failure 400 {object} serverError.ErrorResponse
// @Failure 404 {object} serverError.ErrorResponse
// @Id updatePlaybookTask
// @Router /v1/ansible/playbooks/{playbook_name}/tasks/{task_id} [put]
func (h *PlaybookHandler) HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "task_id")

	var req updateTaskRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	task, err := h.svc.UpdateTask(r.Context(), taskID, UpdateTaskRequest(req))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, errors.New("task not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, updateTaskResponse{Task: task})
}

// HandleDeleteTask deletes a task from a playbook.
// @Summary Delete task
// @Description Removes a task from a playbook
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param playbook_name path string true "Playbook name"
// @Param task_id path string true "Task ID"
// @Success 204
// @Failure 404 {object} serverError.ErrorResponse
// @Id deletePlaybookTask
// @Router /v1/ansible/playbooks/{playbook_name}/tasks/{task_id} [delete]
func (h *PlaybookHandler) HandleDeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "task_id")

	if err := h.svc.DeleteTask(r.Context(), taskID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, errors.New("task not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleReorderTasks reorders tasks in a playbook.
// @Summary Reorder tasks
// @Description Reorders tasks in a playbook
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param playbook_name path string true "Playbook name"
// @Param request body reorderTasksRequest true "New task order"
// @Success 204
// @Failure 400 {object} serverError.ErrorResponse
// @Failure 404 {object} serverError.ErrorResponse
// @Id reorderPlaybookTasks
// @Router /v1/ansible/playbooks/{playbook_name}/tasks/reorder [patch]
func (h *PlaybookHandler) HandleReorderTasks(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "playbook_name")

	pb, err := h.svc.GetPlaybookByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, errors.New("playbook not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	var req reorderTasksRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if len(req.TaskIDs) == 0 {
		serverError.RespondError(w, http.StatusBadRequest, errors.New("task_ids is required"))
		return
	}

	if err := h.svc.ReorderTasks(r.Context(), pb.ID, req.TaskIDs); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleExportPlaybook exports a playbook as YAML.
// @Summary Export playbook
// @Description Exports a playbook as raw YAML
// @Tags Ansible Playbooks
// @Accept json
// @Produce json
// @Param playbook_name path string true "Playbook name"
// @Success 200 {object} exportPlaybookResponse
// @Failure 404 {object} serverError.ErrorResponse
// @Id exportPlaybook
// @Router /v1/ansible/playbooks/{playbook_name}/export [get]
func (h *PlaybookHandler) HandleExportPlaybook(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "playbook_name")

	pb, err := h.svc.GetPlaybookByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, errors.New("playbook not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	yamlContent, err := h.svc.ExportPlaybook(r.Context(), pb.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, err)
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, exportPlaybookResponse{YAML: string(yamlContent)})
}

// RegisterPlaybookRoutes registers playbook routes on the given router.
func (h *PlaybookHandler) RegisterPlaybookRoutes(r chi.Router) {
	r.Route("/playbooks", func(r chi.Router) {
		r.Get("/", h.HandleListPlaybooks)
		r.Post("/", h.HandleCreatePlaybook)

		r.Route("/{playbook_name}", func(r chi.Router) {
			r.Get("/", h.HandleGetPlaybook)
			r.Delete("/", h.HandleDeletePlaybook)
			r.Get("/export", h.HandleExportPlaybook)

			r.Route("/tasks", func(r chi.Router) {
				r.Post("/", h.HandleAddTask)
				r.Patch("/reorder", h.HandleReorderTasks)
				r.Put("/{task_id}", h.HandleUpdateTask)
				r.Delete("/{task_id}", h.HandleDeleteTask)
			})
		})
	})
}
