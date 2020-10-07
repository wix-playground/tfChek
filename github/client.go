package github

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v28/github"
	"github.com/wix-playground/tfChek/misc"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"
)
import "golang.org/x/oauth2"

type Client interface {
	CreatePR(branch string) (*int, error)
	CreateIssue(branch string, assignees *[]string) (*int, error)
	RequestReview(number int, reviewers *[]string) error
	Review(number int, comment string) error
	Close(number int) error
	Comment(number int, comment *string) error
	Merge(number int, message string) (*string, error)
	DeleteBranch(number int) error
	CleanupBranches(before *time.Time, mergedOnly bool) (map[string]bool, error)
	//TODO: add cleanup Issues capability
	//DeleteIssue()
	//CleanupIssues()
	GetArchiveLink(ref string) (*url.URL, error)
}

type ClientRunSH struct {
	Repository string
	Owner      string
	client     *github.Client
	context    context.Context
}

func (c *ClientRunSH) GetArchiveLink(ref string) (*url.URL, error) {
	rcgo := &github.RepositoryContentGetOptions{
		Ref: ref,
	}
	link, response, err := c.client.Repositories.GetArchiveLink(c.context, c.Owner, c.Repository, github.Zipball, rcgo)
	if err != nil {
		var resp string
		body, rerr := ioutil.ReadAll(response.Body)
		if rerr != nil {
			resp = fmt.Sprintf("cannot read response. Error: %s", rerr.Error())
		} else {
			resp = string(body)
		}
		misc.Debugf("failed to get archive download link. Status: %d (%s) Details %s", response.StatusCode, response.Status, resp)
		return nil, fmt.Errorf("failed to get archive link. Error: %w", err)
	}
	return link, nil
}

func wrapComment(data string) *string {
	code := fmt.Sprintf("Command output:\n```%s```", data)
	return &code
}

func (c *ClientRunSH) getHeadSHA(number int) (string, error) {
	pullRequest, _, err := c.client.PullRequests.Get(c.context, c.Owner, c.Repository, number)
	if err != nil {
		log.Printf("Cannot get PR by number %d Error: %s", number, err)
		return "", err
	}
	return *pullRequest.Head.SHA, nil
}

func (c *ClientRunSH) DeleteBranch(number int) error {
	ref := plumbing.NewBranchReferenceName(fmt.Sprintf("%s%d", misc.TaskPrefix, number))
	return c.deleteRef(ref.String())
}

func (c *ClientRunSH) deleteRef(ref string) error {
	response, err := c.client.Git.DeleteRef(c.context, c.Owner, c.Repository, ref)
	if err != nil {
		if response != nil {
			misc.Debugf("Response status %d %s. Body: %s", response.StatusCode, response.Status, response.Body)
		}
		return fmt.Errorf("failed to delete branch %s, Error: %w", ref, err)
	}
	return nil
}

func (c *ClientRunSH) getBranchesList() ([]*github.Reference, error) {
	var result []*github.Reference
	listOptions := &github.ReferenceListOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		refs, response, err := c.client.Git.ListRefs(c.context, c.Owner, c.Repository, listOptions)
		if err != nil {
			if response != nil {
				misc.Debugf("Response status %d %s. Body: %s", response.StatusCode, response.Status, response.Body)
			}
			return result, fmt.Errorf("cannot list branches by prefix %s, Error: %w", misc.TaskPrefix, err)
		}
		//Filter by prefix
		if response.NextPage == 0 || response.LastPage == 0 {
			break
		}
		if refs == nil || len(refs) == 0 {
			break
		}
		for _, ref := range refs {
			if strings.Contains(ref.GetRef(), misc.TaskPrefix) {
				result = append(result, ref)
			}
		}

		listOptions = &github.ReferenceListOptions{ListOptions: github.ListOptions{PerPage: 100, Page: response.NextPage}}
	}
	return result, nil
}

func (c *ClientRunSH) getPRs() ([]*github.PullRequest, error) {
	var result []*github.PullRequest
	listOptions := &github.PullRequestListOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		prs, response, err := c.client.PullRequests.List(c.context, c.Owner, c.Repository, listOptions)
		if err != nil {
			if response != nil {
				misc.Debugf("Response status %d %s. Body: %s", response.StatusCode, response.Status, response.Body)
			}
			return result, fmt.Errorf("cannot list PRs by base prefix %s, Error: %w", misc.TaskPrefix, err)
		}

		if prs != nil && len(prs) > 0 {
			for _, pr := range prs {
				if strings.Contains(*pr.Head.Ref, misc.TaskPrefix) {
					result = append(result, pr)
				}
			}
		}
		if response.NextPage == 0 || response.LastPage == 0 {
			break
		}
		listOptions = &github.PullRequestListOptions{ListOptions: github.ListOptions{PerPage: 100, Page: response.NextPage}}
	}
	return result, nil
}

func (c *ClientRunSH) ensureIssueLabel() (*github.Label, error) {
	var label *github.Label
	label, response, err := c.client.Issues.GetLabel(c.context, c.Owner, c.Repository, misc.IssueLabel)
	if err != nil {
		if response != nil {
			misc.Debugf("Response status %d %s. Body: %s", response.StatusCode, response.Status, response.Body)
		}
		return nil, fmt.Errorf("cannot get label %s, Error: %w", misc.IssueLabel, err)
	}
	if label == nil {
		color := "red"
		labelName := misc.IssueLabel
		labelDescription := misc.IssueLabelDesc
		l := &github.Label{Name: &labelName, Color: &color, Description: &labelDescription}
		label, response, err := c.client.Issues.CreateLabel(c.context, c.Owner, c.Repository, l)
		if err != nil {
			if response != nil {
				misc.Debugf("Response status %d %s. Body: %s", response.StatusCode, response.Status, response.Body)
			}
			return nil, fmt.Errorf("cannot create label %s, Error: %w", misc.IssueLabel, err)
		}
		return label, nil
	}
	return label, nil
}

func (c *ClientRunSH) getIssues() ([]*github.Issue, error) {
	//label, err := c.ensureIssueLabel()
	//if err!= nil {
	//	misc.Debugf("cannot get label for issue query. Error: %s", err)
	//}
	//listOptions := &github.IssueListOptions{Labels: []string{label.String()}}
	listOptions := &github.IssueListByRepoOptions{}
	iss, response, err := c.client.Issues.ListByRepo(c.context, c.Owner, c.Repository, listOptions)
	if err != nil {
		if response != nil {
			misc.Debugf("Response status %d %s. Body: %s", response.StatusCode, response.Status, response.Body)
		}
		return nil, fmt.Errorf("cannot list PRs by base prefix %s, Error: %w", misc.TaskPrefix, err)
	}
	return iss, nil
}

func (c *ClientRunSH) CleanupBranches(before *time.Time, mergedOnly bool) (map[string]bool, error) {
	//cleanup merged PRs
	status := make(map[string]bool)
	prs, err := c.getPRs()
	if err != nil {
		return nil, fmt.Errorf("cannot list PRs. Error: %w", err)
	}
	for i, pr := range prs {
		misc.Debugf("Processing %d/%d PR %d - %q for branch deletion", i+1, len(prs), *pr.ID, *pr.Title)
		ref := pr.Base.GetRef()
		if mergedOnly && !pr.GetMerged() {
			misc.Debugf("Skip not merged PR %d", pr.ID)
			continue
		} else {
			misc.Debugf("abotu to delete branch %s from PR %d", ref, *pr.Title)
		}
		parts := strings.Split(ref, "/")
		branch := parts[len(parts)-1]
		if strings.HasPrefix(branch, misc.TaskPrefix) {
			misc.Debugf("Deleting branch %s", branch)
			err = c.deleteRef(ref)
			if err != nil {
				status[ref] = false
				misc.Debugf("failed to delete branch %s (ref: %s). Error: %s", branch, ref, err)
				//return nil, fmt.Errorf("failed to delete branch %s (ref: %s). Error: %w", branch, ref, err)
			} else {
				status[ref] = true
			}
		} else {
			misc.Debugf("Skip non tfChek related branch %s (ref: %s)", branch, ref)
		}
	}
	if !mergedOnly {
		bList, err := c.getBranchesList()
		if err != nil {
			misc.Debugf("cannot get branches list for repo %s. Error: %e", c.Repository, err)
			return status, fmt.Errorf("cannot get branches list for repo %s. Error: %e", c.Repository, err)
		}
		for i, branch := range bList {
			misc.Debugf("processing ref %d/%d for branch deletion", i, len(bList))
			commit, response, err := c.client.Git.GetCommit(c.context, c.Owner, c.Repository, branch.GetObject().GetSHA())
			if err != nil {
				misc.Debugf("cannot get commit object from ref %s", branch.String())
				if response != nil {
					misc.Debugf("Response status %d %s. Body: %s", response.StatusCode, response.Status, response.Body)
				}
				return status, fmt.Errorf("cannot get commit object by ref %s, Error: %w", branch.String(), err)
			}
			if before == nil {
				misc.Debug("Warning! No time constraint for branch deletion. All branches will be removed")
			}
			if before == nil || commit.GetAuthor().Date.Before(*before) {
				misc.Debugf("deleting %s", branch.GetRef())
				err := c.deleteRef(branch.GetRef())
				if err != nil {
					status[branch.GetRef()] = false
					misc.Debugf("failed to delete %s. Error: %s", branch.String(), err)
					//return fmt.Errorf("failed to delete %s. Error: %w", branch.String(), err)
				}
			}
		}
	}
	return status, nil
}

//Returns merge SHA commit hash and error
func (c *ClientRunSH) Merge(number int, message string) (*string, error) {
	sha, err := c.getHeadSHA(number)
	if err != nil {
		log.Printf("Failed to get SHA of the head commit. Error: %s", err)
		return nil, err
	}
	pro := &github.PullRequestOptions{CommitTitle: message, SHA: sha, MergeMethod: "squash"}
	mergeResult, _, err := c.client.PullRequests.Merge(c.context, c.Owner, c.Repository, number, message, pro)
	if err != nil {
		log.Printf("Cannot merge pull request %d. Error: %s", number, err)
		return nil, err

	}
	return mergeResult.SHA, nil
}

func (c *ClientRunSH) CreateIssue(branch string, assignees *[]string) (*int, error) {
	newIssue := &github.IssueRequest{Title: github.String(fmt.Sprintf("Cannot merge branch %s", branch)),
		Body: github.String("_This pull request was automatically generated by tfChek_\nPlease fix this issue"), Labels: &[]string{misc.IssueLabel}}
	if assignees != nil && len(*assignees) > 0 {
		a := *assignees
		newIssue.Assignee = &a[0]
		//This does not work
		//newIssue.Assignees = assignees
	}
	issue, _, err := c.client.Issues.Create(c.context, c.Owner, c.Repository, newIssue)
	if err != nil {
		log.Printf("Cannot create new issue. Error: %s", err)
		return nil, err
	} else {
		if assignees != nil && len(*assignees) > 0 {
			in := *issue.Number
			issue, _, err = c.client.Issues.AddAssignees(c.context, c.Owner, c.Repository, in, *assignees)
			if err != nil {
				log.Printf("Cannot add assignees to the issue %d. Error: %s", in, err)
			} else {
				log.Printf("Issue %d has been updated with assignees", issue.Number)
			}
		}
	}
	log.Printf("issue has been created %s", issue.GetHTMLURL())
	return issue.Number, nil
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
	log.Printf("PR has been created %s", pr.GetHTMLURL())
	return pr.Number, nil
}

func (c *ClientRunSH) RequestReview(number int, reviewers *[]string) error {
	rr := github.ReviewersRequest{Reviewers: *reviewers}

	_, resp, err := c.client.PullRequests.RequestReviewers(c.context, c.Owner, c.Repository, number, rr)
	if err != nil {

		if ger, ok := err.(*github.ErrorResponse); ok {
			if ger.Message != "Review cannot be requested from pull request author." {
				if resp != nil && resp.Response.StatusCode == 422 && resp.Status == "422 Unprocessable Entity" && err.Error() != "Review cannot be requested from pull request author." {
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
				} else {
					log.Printf("Cannot add reviewer to the pull request. Error: %s", err)
					return err
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
	_, _, err = c.client.PullRequests.SubmitReview(c.context, c.Owner, c.Repository, number, *review.ID, prr)
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
