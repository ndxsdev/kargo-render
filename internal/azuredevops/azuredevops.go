package azuredevops

import (
	"context"
	"fmt"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	gitutil "github.com/akuity/kargo-render/pkg/git"
)

// OpenPR creates a pull request in Azure DevOps
func OpenPR(
	ctx context.Context,
	repoURL string,
	title string,
	description string,
	targetBranch string,
	sourceBranch string,
	creds gitutil.RepoCredentials,
) (string, error) {
	// Parse Azure DevOps URL to extract organization and project
	// Expected format: https://dev.azure.com/{organization}/{project}/_git/{repository}
	urlParts := strings.Split(strings.TrimPrefix(repoURL, "https://dev.azure.com/"), "/")
	if len(urlParts) < 4 {
		return "", fmt.Errorf("invalid Azure DevOps repository URL format")
	}

	organization := urlParts[0]
	project := urlParts[1]
	repository := strings.TrimSuffix(urlParts[3], ".git")

	// Create a connection to Azure DevOps
	connection := azuredevops.NewPatConnection(
		fmt.Sprintf("https://dev.azure.com/%s", organization),
		creds.Password,
	)

	// Get Git client
	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		return "", fmt.Errorf("error creating Azure DevOps Git client: %w", err)
	}

	// Ensure branch names are in the correct format
	sourceBranch = ensureRefFormat(sourceBranch)
	targetBranch = ensureRefFormat(targetBranch)

	// Create pull request
	createPRArgs := git.CreatePullRequestArgs{
		Project: &project,
		GitPullRequestToCreate: &git.GitPullRequest{
			Title:         &title,
			Description:   &description,
			SourceRefName: &sourceBranch,
			TargetRefName: &targetBranch,
			Repository: &git.GitRepository{
				Name: &repository,
			},
		},
	}

	pr, err := gitClient.CreatePullRequest(ctx, createPRArgs)
	if err != nil {
		return "", fmt.Errorf("error creating pull request: %w", err)
	}

	return *pr.Url, nil
}

// ensureRefFormat ensures the branch name is in the correct format for Azure DevOps
// Azure DevOps requires refs/heads/ prefix for branch names
func ensureRefFormat(branchName string) string {
	if !strings.HasPrefix(branchName, "refs/heads/") {
		return "refs/heads/" + strings.TrimPrefix(branchName, "refs/heads/")
	}
	return branchName
}
