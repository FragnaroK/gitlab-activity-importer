package main

import (
	"log"
	"os"
	"time"

	"github.com/furmanp/gitlab-activity-importer/internal"
	"github.com/furmanp/gitlab-activity-importer/internal/services"
)

func main() {
	startNow := time.Now()
	err := internal.CheckEnvVariables()
	if err != nil {
		log.Fatalf("Error during loading environmental variables: %v", err)
	}

	gitlabUser, err := services.GetGitlabUser()

	log.Printf("Fetched user: %v", gitlabUser)
	if err != nil {
		log.Fatalf("Error during reading GitLab User data: %v", err)
	}

	gitLabUserId := gitlabUser.ID

	log.Printf("Fetched user ID: %v", gitLabUserId)
	var projectIds []int
	projectIds, err = services.GetUsersProjectsIds(gitLabUserId)

	log.Printf("Fetched project IDs: %v", projectIds)

	if err != nil {
		log.Fatalf("Error during getting users projects: %v", err)
	}
	if len(projectIds) == 0 {
		log.Print("No contributions found for this user. Closing the program.")
		return
	}

	log.Printf("Found contributions in %v projects \n", len(projectIds))

	repo := services.OpenOrInitClone()

	commitChannel := make(chan []internal.Commit, len(projectIds))

	go func() {
		totalCommits := 0
		log.Println("Starting to import commits.")
		for commits := range commitChannel {
			log.Printf("Importing %v commits.\n", len(commits))
			localCommits := services.CreateLocalCommit(repo, commits)
			totalCommits += localCommits
		}
		log.Printf("Imported %v commits.\n", totalCommits)

	}()

	log.Println("Fetching all commits...")

	services.FetchAllCommits(projectIds, os.Getenv("COMMITER_NAME"), commitChannel)

	log.Println("All commits fetched. Closing the channel.")
	services.PushLocalCommits(repo)
	log.Printf("Operation took: %v in total.", time.Since(startNow))
}
