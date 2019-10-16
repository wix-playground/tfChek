package github

import (
	"context"
	"github.com/google/go-github/v28/github"
	"log"
)
import "golang.org/x/oauth2"

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
		log.Printf("Cannot add reviewer to the pull request. Error: %s\nResponse: %v", err, resp)
		return err
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
	oref := pullRequest.Base.GetRef()
	pullRequest.Base = nil
	review, _, err := c.client.PullRequests.Edit(c.context, c.Owner, c.Repository, number, pullRequest)
	if err != nil {
		log.Printf("Cannot close the pull request %d Error: %s", number, err)
		return err
	}
	cref := review.Base.GetRef()
	log.Printf("Base was %s and become %s", oref, cref)
	log.Printf("PR #%d has been closed %s", number, review.GetHTMLURL())
	return nil
}
