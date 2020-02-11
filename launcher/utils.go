package launcher

import (
	"errors"
	"fmt"
	"github.com/otiai10/copy"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"tfChek/misc"
)

func normalizeGitRemotes(remotes *[]string) *[]string {
	if remotes == nil || len(*remotes) == 0 {
		return nil
	}
	r := *remotes
	var nr *[]string
	//Need to maintain override order here and remove duplicates
	for i := range *remotes {
		nr = prepend2normal(r[len(r)-1-i], nr)
	}
	return nr
}

func prepend2normal(r string, normal *[]string) *[]string {
	var contains bool = false
	if normal == nil {
		return &[]string{r}
	}
	for _, e := range *normal {
		if e == r {
			contains = true
			break
		}
	}
	if !contains {
		n := append([]string{r}, *normal...)
		return &n
	}
	return normal
}

/**
This function copies certificates to landscape directory of run.sh git repo
*/
//TODO: DRY
func deliverLambdas(repo string) error {
	//Actually this has to be automated better than just copy certs from one dir to the repo dir each time it is needed
	//By now this assumes that there must be a predeployed certificate source directory
	lambdaDirectoryPath := strings.TrimSpace(viper.GetString(misc.LambdaSourceKey))
	if lambdaDirectoryPath == "" {
		if viper.GetBool(misc.DebugKey) {
			log.Print("Warning! Lambda source directory is not configured")
		}
		return nil
	}

	landscapePath := repo + string(os.PathSeparator) + "landscape"
	if _, err := os.Stat(landscapePath); os.IsNotExist(err) {
		msg := fmt.Sprintf("Repository %s is missing 'landscape' directory", repo)
		if viper.GetBool(misc.DebugKey) {
			log.Print(msg)
		}
		return errors.New(msg)
	}
	lambdasPath := landscapePath + string(os.PathSeparator) + "lambdas"
	if _, err := os.Stat(lambdasPath); os.IsNotExist(err) {
		err := copy.Copy(lambdaDirectoryPath, lambdasPath)
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Failed to copy directory. Error: %s", err)
			}
		} else {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Lambdas has been copied to the '%s'", lambdasPath)
			}
		}
	} else {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Lambdas check OK! at '%s'", lambdasPath)
		}
	}
	return nil
}

/**
This function copies certificates to landscape directory of run.sh git repo
*/
func deliverCerts(repo string) error {
	//Actually this has to be automated better than just copy certs from one dir to the repo dir each time it is needed
	//By now this assumes that there must be a predeployed certificate source directory
	certDirectoryPath := strings.TrimSpace(viper.GetString(misc.CertSourceKey))
	if certDirectoryPath == "" {
		if viper.GetBool(misc.DebugKey) {
			log.Print("Warning! Certificates source directory is not configured")
		}
		return nil
	}

	landscapePath := repo + string(os.PathSeparator) + "landscape"
	if _, err := os.Stat(landscapePath); os.IsNotExist(err) {
		msg := fmt.Sprintf("Repository %s is missing 'landscape' directory", repo)
		if viper.GetBool(misc.DebugKey) {
			log.Print(msg)
		}
		return errors.New(msg)
	}
	certsPath := landscapePath + string(os.PathSeparator) + "certs"
	if _, err := os.Stat(certsPath); os.IsNotExist(err) {
		err := copy.Copy(certDirectoryPath, certsPath)
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Failed to copy directory. Error: %s", err)
			}
		} else {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Certificates has been copied to the '%s'", certsPath)
			}
		}
	} else {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Certificates check OK! at '%s'", certsPath)
		}
	}
	return nil
}
