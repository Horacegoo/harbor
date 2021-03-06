package api

import (
	"fmt"

	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/rbac"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/pkg/notification"
)

// NotificationJobAPI ...
type NotificationJobAPI struct {
	BaseController
	project *models.Project
}

// Prepare ...
func (w *NotificationJobAPI) Prepare() {
	w.BaseController.Prepare()
	if !w.SecurityCtx.IsAuthenticated() {
		w.SendUnAuthorizedError(errors.New("UnAuthorized"))
		return
	}

	pid, err := w.GetInt64FromPath(":pid")
	if err != nil {
		w.SendBadRequestError(fmt.Errorf("failed to get project ID: %v", err))
		return
	}
	if pid <= 0 {
		w.SendBadRequestError(fmt.Errorf("invalid project ID: %d", pid))
		return
	}

	project, err := w.ProjectCtl.Get(w.Context(), pid)
	if err != nil {
		if errors.IsNotFoundErr(err) {
			w.SendNotFoundError(fmt.Errorf("project %d not found", pid))
		} else {
			w.SendInternalServerError(fmt.Errorf("failed to get project %d: %v", pid, err))
		}
		return
	}

	w.project = project
}

// List ...
func (w *NotificationJobAPI) List() {
	if !w.validateRBAC(rbac.ActionList, w.project.ProjectID) {
		return
	}

	policyID, err := w.GetInt64("policy_id")
	if err != nil || policyID <= 0 {
		w.SendBadRequestError(fmt.Errorf("invalid policy_id: %s", w.GetString("policy_id")))
		return
	}

	policy, err := notification.PolicyMgr.Get(policyID)
	if err != nil {
		w.SendInternalServerError(fmt.Errorf("failed to get policy %d: %v", policyID, err))
		return
	}
	if policy == nil {
		w.SendBadRequestError(fmt.Errorf("policy %d not found", policyID))
		return
	}

	query := &models.NotificationJobQuery{
		PolicyID: policyID,
	}

	query.Statuses = w.GetStrings("status")

	query.Page, query.Size, err = w.GetPaginationParams()
	if err != nil {
		w.SendBadRequestError(err)
		return
	}

	total, jobs, err := notification.JobMgr.List(query)
	if err != nil {
		w.SendInternalServerError(fmt.Errorf("failed to list notification jobs: %v", err))
		return
	}
	w.SetPaginationHeader(total, query.Page, query.Size)
	w.WriteJSONData(jobs)
}

func (w *NotificationJobAPI) validateRBAC(action rbac.Action, projectID int64) bool {
	if w.SecurityCtx.IsSysAdmin() {
		return true
	}

	return w.RequireProjectAccess(projectID, action, rbac.ResourceNotificationPolicy)
}
