package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
	"tfChek/github"
	"tfChek/misc"
)

type DeleteResponse struct {
	Error  error
	Status map[string]*DeleteStatus
}

type DeleteStatus struct {
	Deleted bool
	Error   error
}

func DeleteCIBranch(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	branch, bok := v[misc.ApiBranchKey]
	id, iok := v[misc.IdParam]
	managers := github.GetAllManagers()
	var err error
	var taskId int
	if bok {
		s := strings.Split(branch, "-")
		if len(s) != 2 {
			misc.Debugf("failed to split branch %s name onto 2 parts. Falling through to next parameter")
		} else {
			taskId, err = strconv.Atoi(s[1])
			if err != nil {
				misc.Debugf("cannot convert branch id %s to int. Error: %s", s[1], err)
				w.WriteHeader(http.StatusNotAcceptable)
				dr := &DeleteResponse{Error: fmt.Errorf("cannot convert branch %s to int. Error: %w", branch, err)}
				mdr, err := json.Marshal(dr)
				if err != nil {
					w.Header().Add(misc.ContentTypeKey, misc.ContentTypeMarkdown)
					misc.Debugf("cannot marshal response %v. Falling back to a plain text message. Error: %s", dr, err)
					_, err := w.Write([]byte(fmt.Sprintf("cannot convert branch %s to int. Error: %s", branch, err)))
					if err != nil {
						misc.Debugf("cannot send a response. Error: %s", err)
					}
					return
				}
				_, err = w.Write(mdr)
				if err != nil {
					misc.Debugf("cannot send a response. Error: %s", err)
				}
				return
			}
		}

	}
	if iok {
		taskId, err = strconv.Atoi(id)
		if err != nil {
			misc.Debugf("cannot convert branch id %s to int. Error: %s", id, err)
			w.WriteHeader(http.StatusNotAcceptable)
			dr := &DeleteResponse{Error: fmt.Errorf("cannot convert branch id %s to int. Error: %w", id, err)}
			mdr, err := json.Marshal(dr)
			if err != nil {
				w.Header().Add(misc.ContentTypeKey, misc.ContentTypeMarkdown)
				misc.Debugf("cannot marshal response %v. Falling back to a plain text message. Error: %s", dr, err)
				_, err := w.Write([]byte(fmt.Sprintf("cannot convert branch id %s to int. Error: %s", id, err)))
				if err != nil {
					misc.Debugf("cannot send a response. Error: %s", err)
				}
				return
			}
			_, err = w.Write(mdr)
			if err != nil {
				misc.Debugf("cannot send a response. Error: %s", err)
			}
			return
		}
		branch = fmt.Sprintf("%s%s", misc.TaskPrefix, id)
		misc.Debugf("%s=%s parameter has been provided in request, this is why %s parameter has been overridden to %s", misc.IdParam, id, misc.ApiBranchKey, branch)
	}

	//Actual logic is here
	status := make(map[string]*DeleteStatus)
	for _, m := range managers {
		err := m.GetClient().DeleteBranch(taskId)
		if err != nil {
			status[m.Repository] = &DeleteStatus{Deleted: false, Error: err}
			misc.Debugf("failed to delete branch %s in repo %s. Error: %s", branch, m.Repository, err)
		}
	}
	ds := &DeleteResponse{Error: nil, Status: status}
	marshalledStatus, err := json.Marshal(ds)
	w.WriteHeader(http.StatusAccepted)
	if err != nil {
		misc.Debugf("cannot marshal response %v. Error: %s", ds, err)
		w.Header().Add(misc.ContentTypeKey, misc.ContentTypeMarkdown)

		_, err := w.Write([]byte(fmt.Sprintf("cannot marshal response %v. Falling back to text. Error: %s", ds, err)))
		if err != nil {
			misc.Debugf("cannot send a response. Error: %s", err)
		}
		return
	}
	w.Header().Add(misc.ContentTypeKey, misc.ContentTypeJson)
	_, err = w.Write(marshalledStatus)
	if err != nil {
		misc.Debugf("cannot send a response %v. Error: %s", marshalledStatus, err)
	}
}
