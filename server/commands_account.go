package main

import (
	"fmt"

	"github.com/jszwedko/go-circleci"

	"github.com/mattermost/mattermost-server/v5/model"
)

func (p *Plugin) executeMe(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	token, exists := p.getTokenFromKVStore(args.UserId)
	if !exists {
		return p.sendEphemeralResponse(args, notConnectedText), nil
	}

	user, ok := p.getCircleCIUserInfo(token)
	if !ok {
		return p.sendEphemeralResponse(args, errorConnectionText), nil
	}

	circleciClient := &circleci.Client{Token: token}
	projects, _ := circleciClient.ListProjects()
	projectsListString := ""
	for _, project := range projects {
		projectsListString += fmt.Sprintf("- [%s](%s) owned by %s\n", project.Reponame, project.VCSURL, project.Username)
	}

	_ = p.sendEphemeralPost(
		args,
		"",
		[]*model.SlackAttachment{
			{
				ThumbURL: user.AvatarURL,
				Fallback: "User:" + getFormattedNameAndLogin(user) + ". Email:" + *user.SelectedEmail,
				Pretext:  "Information for CircleCI user " + getFormattedNameAndLogin(user),
				Fields: []*model.SlackAttachmentField{
					{
						Title: "Name",
						Value: user.Name,
						Short: true,
					},
					{
						Title: "Email",
						Value: user.SelectedEmail,
						Short: true,
					},
					{
						Title: "Followed projects",
						Value: projectsListString,
						Short: false,
					},
				},
			},
		},
	)

	return &model.CommandResponse{}, nil
}

func (p *Plugin) executeConnect(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	if len(split) < 1 {
		return p.sendEphemeralResponse(args, "Please tell me your token. If you don't have a CircleCI Personal API Token, you can get one from your [Account Dashboard](https://circleci.com/account/api)"), nil
	}

	if token, exists := p.getTokenFromKVStore(args.UserId); exists {
		user, ok := p.getCircleCIUserInfo(token)
		if !ok {
			return p.sendEphemeralResponse(args, "Internal error when reaching CircleCI"), nil
		}

		return p.sendEphemeralResponse(args, "You are already connected as "+getFormattedNameAndLogin(user)), nil
	}

	circleciToken := split[0]
	circleciClient := &circleci.Client{
		Token: circleciToken,
	}

	user, err := circleciClient.Me()
	if err != nil {
		p.API.LogError("Error when reaching CircleCI", "CircleCI error:", err)
		return p.sendEphemeralResponse(args, "Can't connect to CircleCI. Please check that your user API token is valid"), nil
	}

	if ok := p.storeTokenInKVStore(args.UserId, circleciToken); !ok {
		return p.sendEphemeralResponse(args, "Internal error when storing your token"), nil
	}

	return p.sendEphemeralResponse(args, "Successfully connected to CircleCI as "+getFormattedNameAndLogin(user)), nil
}

func (p *Plugin) executeDisconnect(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if _, exists := p.getTokenFromKVStore(args.UserId); !exists {
		return p.sendEphemeralResponse(args, notConnectedText), nil
	}

	if ok := p.deleteTokenFromKVStore(args.UserId); !ok {
		return p.sendEphemeralResponse(args, errorConnectionText), nil
	}

	return p.sendEphemeralResponse(args, "Your CircleCI account has been successfully disconnected from Mattermost"), nil
}