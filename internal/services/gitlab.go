package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/furmanp/gitlab-activity-importer/internal"
)

func GetGitlabUser() (internal.GitLabUser, error) {
	url := os.Getenv("BASE_URL")

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%v/api/v4/user", url), nil)
	if err != nil {
		return internal.GitLabUser{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))

	log.Printf("Fetching user from %v", url)

	res, err := client.Do(req)
	if err != nil {
		return internal.GitLabUser{}, fmt.Errorf("error making the request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return internal.GitLabUser{}, fmt.Errorf("request failed with status code: %v", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return internal.GitLabUser{}, fmt.Errorf("error reading the response body: %v", err)
	}

	var user internal.GitLabUser
	if err := json.Unmarshal(body, &user); err != nil {
		return internal.GitLabUser{}, fmt.Errorf("error parsing JSON: %v", err)
	}

	log.Printf("Fetched user: %v", user)
	return user, nil
}

func GetUsersProjectsIds(userId int) ([]int, error) {
	url := os.Getenv("BASE_URL")

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%v/api/v4/users/%v/contributed_projects", url, userId), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating the request: %v", err)
	}

	req.Header.Set("PRIVATE-TOKEN", os.Getenv("GITLAB_TOKEN"))
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making the request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code: %v", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var projects []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	projectIds := make([]int, len(projects))
	for i, project := range projects {
		projectIds[i] = project.ID
	}

	return projectIds, nil
}

func GetProjectCommits(projectId int, userName string) ([]internal.Commit, error) {
	url := os.Getenv("BASE_URL")
	token := os.Getenv("GITLAB_TOKEN")

    var allCommits []internal.Commit
    client := &http.Client{}
    page := 1
    maxPages := 50 // Set a reasonable limit
    
    seenCommits := make(map[string]bool) // Track commits we've already seen

	for page <= maxPages {
		req, err := http.NewRequest("GET", fmt.Sprintf("%v/api/v4/projects/%v/repository/commits?author=%v&per_page=100&page=%d", url, projectId, userName, page), nil)
		if err != nil {
			return nil, fmt.Errorf("error fetching the commits: %v", err)
		}

		req.Header.Set("PRIVATE-TOKEN", token)
		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making the request: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed with status code: %v", res.StatusCode)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading the response body: %v", err)
		}

		var commits []internal.Commit
        err = json.Unmarshal(body, &commits)
        if err != nil {
            return nil, fmt.Errorf("error parsing JSON: %v", err)
        }
        
        if len(commits) == 0 {
            break // No more commits
        }
        
        // Check for duplicate pages
        newCommitsFound := false
        for _, commit := range commits {
            if !seenCommits[commit.ID] {
                seenCommits[commit.ID] = true
                allCommits = append(allCommits, commit)
                newCommitsFound = true
            }
        }
        
        if !newCommitsFound {
            log.Printf("No new commits found on page %d for project %d, stopping pagination", page, projectId)
            break // Exit if we're just getting duplicates
        }
        
        page++
	}

	if page > maxPages {
        log.Printf("Warning: Reached maximum page limit (%d) for project %d", maxPages, projectId)
    }

	if len(allCommits) == 0 {
		return nil, fmt.Errorf("found no commits in project no.:%v", projectId)
	}

	log.Printf("Found total of %v commits in project no.:%v \n", len(allCommits), projectId)

	return allCommits, nil
}

func FetchAllCommits(projectIds []int, commiterName string, commitChannel chan []internal.Commit) {
	var wg sync.WaitGroup

	for _, projectId := range projectIds {
		wg.Add(1)
		log.Printf("Fetching commits for project %d", projectId)
		go func(projId int) {
			defer wg.Done()

			commits, err := GetProjectCommits(projId, commiterName)
			if err != nil {
				log.Printf("Error fetching commits for project %d: %v", projId, err)
				return
			}
			if len(commits) > 0 {
				log.Printf("Fetched %v commits from project no.:%v \n", len(commits), projId)
				commitChannel <- commits
			}

		}(projectId)
	}

	log.Println("Waiting for all commits to be fetched")
	wg.Wait()
	log.Println("All commits fetched")
	close(commitChannel)
	log.Println("Channel closed")
}
