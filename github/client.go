package github

import (
	"context"
	"github.com/google/go-github/v28/github"
	"log"
)
import "golang.org/x/oauth2"

type GitHubClient interface {
	CreatePR(branch string) (*int, error)
	RequestReview(number int, reviewers *[]string) error
	Review(number int, comment string) error
	Close(number int) error
	Comment(number int, comment *string) error
}

type ClientRunSH struct {
	Repository string
	Owner      string
	client     *github.Client
	context    context.Context
}

func NewClientRunSH(repository, owner, token string) *ClientRunSH {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	c := ClientRunSH{Repository: repository, Owner: owner, client: client, context: ctx}
	return &c
}

func (c *ClientRunSH) Comment(number int, comment *string) error {
	cmnt := &github.IssueComment{Body: comment}
	requestComment, response, err := c.client.Issues.CreateComment(c.context, c.Owner, c.Repository, number, cmnt)
	if err != nil {
		log.Printf("Cannot comment a pull request number %d Error: %s", number, err)
		if requestComment != nil {
			log.Printf("PR Comment id: %d failed. Response status %s", *requestComment.ID, response.Status)
		}
		return err
	}
	return nil
}

func (c *ClientRunSH) CreatePR(branch string) (*int, error) {

	newPR := &github.NewPullRequest{Title: github.String("Automatic"),
		Head:                github.String(branch),
		Base:                github.String("master"),
		Body:                github.String("tfChek generated pull request"),
		MaintainerCanModify: github.Bool(true)}
	pr, _, err := c.client.PullRequests.Create(c.context, c.Owner, c.Repository, newPR)
	if err != nil {
		log.Printf("Cannot create new pull request. Error: %s", err)
		return nil, err
	}
	log.Printf("PR created %s", pr.GetHTMLURL())
	return pr.Number, nil
}

func (c *ClientRunSH) RequestReview(number int, reviewers *[]string) error {
	rr := github.ReviewersRequest{Reviewers: *reviewers}

	_, resp, err := c.client.PullRequests.RequestReviewers(c.context, c.Owner, c.Repository, number, rr)
	if err != nil {

		if ger, ok := err.(*github.ErrorResponse); ok {
			if ger.Message != "Review cannot be requested from pull request author." {
				if resp.Response.StatusCode == 422 && resp.Status == "422 Unprocessable Entity" && err.Error() != "Review cannot be requested from pull request author." {
					log.Println("Trying to add user as a collaborator")
					repository, _, err := c.client.Repositories.Get(c.context, c.Owner, c.Repository)
					if err != nil {
						log.Printf("Cannot fetch repo. Error: %s", err)
					}
					opts := github.RepositoryAddCollaboratorOptions{Permission: "admin"}
					for _, rv := range *reviewers {
						u, _, err := c.client.Users.Get(c.context, rv)
						if err != nil {
							log.Printf("cannot find user %s, Error: %s", rv, err)
						}
						resp, err = c.client.Repositories.AddCollaborator(c.context, *repository.Owner.Login, *repository.Name, *u.Login, &opts)
						if err != nil {
							log.Printf("Cannot add user %s as a collaborator. Error %s\nResponse: %v", rv, err, resp)
						}
					}
					_, resp, err := c.client.PullRequests.RequestReviewers(c.context, c.Owner, c.Repository, number, rr)
					if err != nil {
						log.Printf("Cannot add reviewer to the pull request. Error: %s\nResponse: %v", err, resp)
						return err
					}
				}
			} else {
				log.Printf("Cannot add reviewer to the pull request. Error: %s\nResponse: %v", err, resp)
				return err
			}
		} else {
			log.Printf("Cannot add reviewer to the pull request. Error: %s\nResponse: %v", err, resp)
			return err
		}
	}
	return nil
}

func (c *ClientRunSH) Review(number int, comment string) error {
	prc := &github.PullRequestReviewRequest{Body: &comment}
	prr := &github.PullRequestReviewRequest{Event: github.String("APPROVE")}
	review, _, err := c.client.PullRequests.CreateReview(c.context, c.Owner, c.Repository, number, prc)
	if err != nil {
		log.Printf("Cannot create review of the pull request %d Error: %s", number, err)
		return err
	}
	c.client.PullRequests.SubmitReview(c.context, c.Owner, c.Repository, number, *review.ID, prr)
	if err != nil {
		log.Printf("Cannot submit review of the pull request %d Error: %s", number, err)
		return err
	}
	log.Printf("PR #%d reviewed %s", number, review.GetHTMLURL())
	return nil
}

func (c *ClientRunSH) Close(number int) error {
	pullRequest, _, err := c.client.PullRequests.Get(c.context, c.Owner, c.Repository, number)
	if err != nil {
		log.Printf("Cannot get the pull request %d Error: %s", number, err)
		return err
	}
	pullRequest.State = github.String("closed")
	pullRequest.Base = nil
	review, _, err := c.client.PullRequests.Edit(c.context, c.Owner, c.Repository, number, pullRequest)
	if err != nil {
		log.Printf("Cannot close the pull request %d Error: %s", number, err)
		return err
	}
	log.Printf("PR #%d has been closed %s", number, review.GetHTMLURL())
	return nil
}
