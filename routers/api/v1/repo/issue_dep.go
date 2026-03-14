// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"

	issues_model "code.gitea.io/gitea/models/issues"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
)

// GetIssueDependencies lists dependencies for an issue
func GetIssueDependencies(ctx *context.APIContext) {
	if !setting.IssueGraphSettings.Enabled {
		ctx.APIErrorNotFound("Issue graph features are disabled")
		return
	}

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.PathParamInt64("index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.APIErrorNotFound("issue not found")
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	deps, err := issues_model.GetDependenciesByIssueID(ctx, issue.ID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	apiIssues := make([]*api.Issue, 0, len(deps))
	for _, dep := range deps {
		if err := dep.LoadRepo(ctx); err != nil {
			ctx.APIErrorInternal(err)
			return
		}
		apiIssues = append(apiIssues, convert.ToAPIIssue(ctx, ctx.Doer, dep))
	}

	ctx.JSON(http.StatusOK, apiIssues)
}

// CreateIssueDependency creates a dependency
func CreateIssueDependency(ctx *context.APIContext) {
	if !setting.IssueGraphSettings.Enabled {
		ctx.APIErrorNotFound("Issue graph features are disabled")
		return
	}

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.PathParamInt64("index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.APIErrorNotFound("issue not found")
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	form := web.GetForm(ctx).(*api.IssueMeta)
	dep, err := resolveDependencyIssue(ctx, form)
	if err != nil {
		return // error already written to ctx
	}

	if dep.ID == issue.ID {
		ctx.APIError(http.StatusBadRequest, "cannot add dependency on itself")
		return
	}

	if err := issues_model.CreateIssueDependency(ctx, ctx.Doer, issue, dep); err != nil {
		if issues_model.IsErrDependencyExists(err) {
			ctx.APIError(http.StatusConflict, "dependency already exists")
			return
		} else if issues_model.IsErrCircularDependency(err) {
			ctx.APIError(http.StatusBadRequest, "would create circular dependency")
			return
		}
		ctx.APIErrorInternal(err)
		return
	}

	ctx.Status(http.StatusCreated)
}

// RemoveIssueDependency removes a dependency
func RemoveIssueDependency(ctx *context.APIContext) {
	if !setting.IssueGraphSettings.Enabled {
		ctx.APIErrorNotFound("Issue graph features are disabled")
		return
	}

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.PathParamInt64("index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.APIErrorNotFound("issue not found")
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	form := web.GetForm(ctx).(*api.IssueMeta)
	dep, err := resolveDependencyIssue(ctx, form)
	if err != nil {
		return
	}

	if err := issues_model.RemoveIssueDependency(ctx, ctx.Doer, issue, dep, issues_model.DependencyTypeBlockedBy); err != nil {
		if issues_model.IsErrDependencyNotExists(err) {
			ctx.APIError(http.StatusNotFound, "dependency not found")
			return
		}
		ctx.APIErrorInternal(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetIssueBlocks lists issues that this issue blocks
func GetIssueBlocks(ctx *context.APIContext) {
	if !setting.IssueGraphSettings.Enabled {
		ctx.APIErrorNotFound("Issue graph features are disabled")
		return
	}

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.PathParamInt64("index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.APIErrorNotFound("issue not found")
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	blocks, err := issues_model.GetBlockedByDependencies(ctx, issue.ID)
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	apiIssues := make([]*api.Issue, 0, len(blocks))
	for _, blocked := range blocks {
		if err := blocked.LoadRepo(ctx); err != nil {
			ctx.APIErrorInternal(err)
			return
		}
		apiIssues = append(apiIssues, convert.ToAPIIssue(ctx, ctx.Doer, blocked))
	}

	ctx.JSON(http.StatusOK, apiIssues)
}

// CreateIssueBlocking creates a blocking relationship (this issue blocks another)
func CreateIssueBlocking(ctx *context.APIContext) {
	if !setting.IssueGraphSettings.Enabled {
		ctx.APIErrorNotFound("Issue graph features are disabled")
		return
	}

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.PathParamInt64("index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.APIErrorNotFound("issue not found")
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	form := web.GetForm(ctx).(*api.IssueMeta)
	blocked, err := resolveDependencyIssue(ctx, form)
	if err != nil {
		return
	}

	if blocked.ID == issue.ID {
		ctx.APIError(http.StatusBadRequest, "cannot block itself")
		return
	}

	// Create: blocked depends on issue (issue blocks blocked)
	if err := issues_model.CreateIssueDependency(ctx, ctx.Doer, blocked, issue); err != nil {
		if issues_model.IsErrDependencyExists(err) {
			ctx.APIError(http.StatusConflict, "blocking relationship already exists")
			return
		} else if issues_model.IsErrCircularDependency(err) {
			ctx.APIError(http.StatusBadRequest, "would create circular dependency")
			return
		}
		ctx.APIErrorInternal(err)
		return
	}

	ctx.Status(http.StatusCreated)
}

// RemoveIssueBlocking removes a blocking relationship
func RemoveIssueBlocking(ctx *context.APIContext) {
	if !setting.IssueGraphSettings.Enabled {
		ctx.APIErrorNotFound("Issue graph features are disabled")
		return
	}

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.PathParamInt64("index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.APIErrorNotFound("issue not found")
		} else {
			ctx.APIErrorInternal(err)
		}
		return
	}

	form := web.GetForm(ctx).(*api.IssueMeta)
	blocked, err := resolveDependencyIssue(ctx, form)
	if err != nil {
		return
	}

	if err := issues_model.RemoveIssueDependency(ctx, ctx.Doer, blocked, issue, issues_model.DependencyTypeBlocking); err != nil {
		if issues_model.IsErrDependencyNotExists(err) {
			ctx.APIError(http.StatusNotFound, "blocking relationship not found")
			return
		}
		ctx.APIErrorInternal(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// resolveDependencyIssue resolves an IssueMeta to an Issue, checking permissions.
func resolveDependencyIssue(ctx *context.APIContext, form *api.IssueMeta) (*issues_model.Issue, error) {
	// Determine the repo for the dependency
	depRepo := ctx.Repo.Repository

	if form.Owner != "" && form.Name != "" {
		// Cross-repo dependency
		if !setting.Service.AllowCrossRepositoryDependencies {
			ctx.APIError(http.StatusBadRequest, "cross-repository dependencies are not enabled")
			return nil, fmt.Errorf("cross-repository dependencies disabled")
		}

		var err error
		depRepo, err = repo_model.GetRepositoryByOwnerAndName(ctx, form.Owner, form.Name)
		if err != nil {
			if repo_model.IsErrRepoNotExist(err) {
				ctx.APIErrorNotFound("repository not found")
			} else {
				ctx.APIErrorInternal(err)
			}
			return nil, err
		}

		perm, err := access_model.GetUserRepoPermission(ctx, depRepo, ctx.Doer)
		if err != nil {
			ctx.APIErrorInternal(err)
			return nil, err
		}
		if !perm.CanReadIssuesOrPulls(false) {
			ctx.APIErrorNotFound("repository not found")
			return nil, fmt.Errorf("no permission to read issues")
		}
	}

	dep, err := issues_model.GetIssueByIndex(ctx, depRepo.ID, form.Index)
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.APIErrorNotFound("dependency issue not found")
		} else {
			ctx.APIErrorInternal(err)
		}
		return nil, err
	}

	return dep, nil
}
