package plugin

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/nathanaelhoun/mattermost-plugin-circleci/server/circle"
)

func (p *Plugin) httpHandleApprove(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	circleciToken, exists := p.Store.GetTokenForUser(userID, p.getConfiguration().EncryptionKey)
	if !exists {
		http.NotFound(w, r)
	}
	requestData := model.PostActionIntegrationRequestFromJson(r.Body)
	if requestData == nil {
		p.API.LogError("Empty request data")
		p.sendEphemeralResponse(&model.CommandArgs{}, "Cannot approve the workflow from mattermost. Please go [here](http://app.circleci.com)")
		return
	}

	workflowID := fmt.Sprintf("%v", requestData.Context["WorkflowID"])
	jobs, err := circle.GetWorkflowJobs(circleciToken, workflowID)

	if err != nil {
		p.API.LogError("Error occurred while getting workflow jobs", err)
		// TODO: replace with actual workflow URL to approve in circle as a fallback
		p.sendEphemeralResponse(&model.CommandArgs{}, "Cannot approve the workflow from mattermost. Please go [here](http://app.circleci.com)")
		return
	}

	var approvalRequestID string
	for _, job := range *jobs {
		if job.ApprovalRequestId != "" {
			p.API.LogDebug("Job with Approval", "request Id ", job.Id)
			approvalRequestID = fmt.Sprintf("%v", job.ApprovalRequestId)
			break
		}
	}
	_, err = circle.ApproveJob(circleciToken, approvalRequestID, workflowID)

	responsePost := &model.Post{
		ChannelId: requestData.ChannelId,
		RootId:    requestData.PostId,
		UserId:    p.botUserID,
	}

	// TODO update the original post to remove the button

	if err != nil {
		p.API.LogError("Error occurred while approving", err)
		// TODO: replace with actual workflow URL to approve in circle as a fallback
		responsePost.Message = "Cannot approve the workflow from mattermost. Please go [here](http://app.circleci.com)"
	} else {
		responsePost.Message = "Workflow successfully approved :+1:"
	}

	_, appErr := p.API.CreatePost(responsePost)
	if appErr != nil {
		p.API.LogError("Error when creating post", "appError", appErr)
	}
}
