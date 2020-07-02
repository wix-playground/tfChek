package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
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
	var taskId int
	if iok {
		taskId, err := strconv.Atoi(id)

		if err != nil {
			misc.Debugf("cannot convert branch id %s to int. Error: %s", id, err)
			w.WriteHeader(http.StatusNotAcceptable)
			//TODO: return Status here
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
	//TODO: implement branch aram too
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
