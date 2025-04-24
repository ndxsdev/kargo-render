package azuredevops

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	gitutil "github.com/akuity/kargo-render/pkg/git"
)

// parseAzureDevOpsURL parses an Azure DevOps repository URL and returns organization, project, and repository names
func parseAzureDevOpsURL(repoURL string) (org, proj, repo string, err error) {
	if strings.Contains(repoURL, "dev.azure.com") {
		urlParts := strings.Split(strings.TrimPrefix(repoURL, "https://dev.azure.com/"), "/")
		if len(urlParts) < 4 {
			return "", "", "", fmt.Errorf("invalid Azure DevOps repository URL format")
		}
		org = urlParts[0]
		proj = urlParts[1]
		repo = strings.TrimSuffix(urlParts[3], ".git")
	} else if strings.Contains(repoURL, ".visualstudio.com") {
		urlParts := strings.Split(repoURL, "/")
		if len(urlParts) < 5 {
			return "", "", "", fmt.Errorf("invalid Azure DevOps repository URL format")
		}
		org = strings.Split(urlParts[2], ".")[0]
		proj = urlParts[3]
		repo = strings.TrimSuffix(urlParts[5], ".git")
	} else {
		return "", "", "", fmt.Errorf("unsupported Azure DevOps repository URL format")
	}
	return org, proj, repo, nil
}

// getRepositoryID gets the repository ID from Azure DevOps
func getRepositoryID(ctx context.Context, client git.Client, project, repository string) (*uuid.UUID, error) {
	repos, err := client.GetRepositories(ctx, git.GetRepositoriesArgs{
		Project: &project,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing repositories: %w", err)
	}

	if repos != nil {
		for _, repo := range *repos {
			if *repo.Name == repository {
				return repo.Id, nil
			}
		}
	}

	return nil, fmt.Errorf("repository '%s' not found in project '%s'", repository, project)
}

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
	// Ensure we have a PAT token as password
	if creds.Password == "" {
		return "", fmt.Errorf("Azure DevOps requires a Personal Access Token (PAT) as password")
	}

	// Parse Azure DevOps URL
	organization, project, repository, err := parseAzureDevOpsURL(repoURL)
	if err != nil {
		return "", err
	}

	// Create a connection to Azure DevOps
	connection := azuredevops.NewPatConnection(
		fmt.Sprintf("https://dev.azure.com/%s", organization),
		creds.Password,
	)

	// Create Git client
	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		return "", fmt.Errorf("error creating Azure DevOps Git client: %w", err)
	}

	// Get repository ID
	repoUUID, err := getRepositoryID(ctx, gitClient, project, repository)
	if err != nil {
		return "", err
	}

	// Ensure branch names are in the correct format
	sourceBranch = ensureRefFormat(sourceBranch)
	targetBranch = ensureRefFormat(targetBranch)

	// Create pull request
	createPRArgs := git.CreatePullRequestArgs{
		Project: &project,
		RepositoryId: repoUUID,
		GitPullRequestToCreate: &git.GitPullRequest{
			Title:         &title,
			Description:   &description,
			SourceRefName: &sourceBranch,
			TargetRefName: &targetBranch,
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
