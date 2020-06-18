package launcher

import (
	"errors"
	"time"
)

type RunShCmd struct {
	All              bool
	Omit             bool
	No               bool
	Yes              bool
	Debug            bool
	Env              string
	Layer            string
	Targets          []string
	UsePlan          bool
	Filter           string
	Region           string
	TerraformVersion string
	hash             string
	GitOrigins       []string
	Started          *time.Time
}

func (rsc *RunShCmd) CommandArgs() (string, []string, error) {
	command := "./run.sh"
	var args []string
	if rsc.All {
		args = append(args, "-a")
	}
	if rsc.Omit {
		args = append(args, "-o")
	}
	if rsc.No {
		args = append(args, "-n")
	}
	if rsc.Yes && !rsc.No {
		args = append(args, "-y")
	}
	if rsc.UsePlan {
		args = append(args, "-p")
	}
	if rsc.Debug {
		args = append(args, "-d")
	}
	if rsc.Region != "" {
		args = append(args, "-r", rsc.Region)
	}
	//Commented out because flag -t conflicts with timeout flag
	//if rsc.TerraformVersion != "" {
	//	args = append(args, "-t", rsc.TerraformVersion)
	//}
	if rsc.Env != "" {
		if rsc.Layer != "" {
			args = append(args, rsc.Env+"/"+rsc.Layer)
		} else {
			args = append(args, rsc.Env)
		}
	} else {
		return "", nil, errors.New("cannot launch run.sh if environment is not specified")
	}
	return command, args, nil
}

func (rsc *RunShCmd) getGitOrigins() *[]string {
	return &rsc.GitOrigins
}
