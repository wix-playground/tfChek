package git

import (
	"fmt"
	"regexp"
)

func GetFullRepoName(gitUrl string) (string, error) {
	sshUrlRE, err := regexp.Compile("git@.*:(.*).git")
	if err != nil {
		return "", fmt.Errorf("cannot compile regex for SSH Git URL %w", err)
	}
	gitUrlRE, err := regexp.Compile("git://.[^/]+/(.*).git")
	if err != nil {
		return "", fmt.Errorf("cannot compile regex for Git URL %w", err)
	}
	cloneUrlRE, err := regexp.Compile("https://.[^/]+/(.*).git")
	if err != nil {
		return "", fmt.Errorf("cannot compile regex for Git clone URL %w", err)
	}
	httpsUrlRE, err := regexp.Compile("https://.[^/]+/(.*)")
	if err != nil {
		return "", fmt.Errorf("cannot compile regex for https Git URL %w", err)
	}
	if sshUrlRE.MatchString(gitUrl) {
		matches := sshUrlRE.FindStringSubmatch(gitUrl)
		return matches[1], nil
	}
	if gitUrlRE.MatchString(gitUrl) {
		matches := gitUrlRE.FindStringSubmatch(gitUrl)
		return matches[1], nil
	}
	if cloneUrlRE.MatchString(gitUrl) {
		matches := cloneUrlRE.FindStringSubmatch(gitUrl)
		return matches[1], nil
	}
	if httpsUrlRE.MatchString(gitUrl) {
		matches := httpsUrlRE.FindStringSubmatch(gitUrl)
		return matches[1], nil
	}
	return "", fmt.Errorf("No known URL schemas matched provided url")
}
