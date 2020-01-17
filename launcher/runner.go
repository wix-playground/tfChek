package launcher

import (
	"errors"
)

type RunShCmd struct {
	All     bool
	Omit    bool
	No      bool
	Yes     bool
	Env     string
	Layer   string
	Targets []string
}

func (rsc *RunShCmd) CommandArgs() (string, []string, error) {
	//TODO: implement all command arguments form run.sh script
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
