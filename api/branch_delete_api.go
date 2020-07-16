package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/wix-system/tfChek/github"
	"github.com/wix-system/tfChek/misc"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//TODO: introduce some interface here for error reporting
type DeleteResponse struct {
	Error    error  `json:"-"`
	ErrorMsg string `json:"error"`
	Status   map[string]*DeleteStatus
}

type DeleteStatus struct {
	Status   map[string]bool
	Error    error  `json:"-"`
	ErrorMsg string `json:"error"`
}

type CleanupForm struct {
	Before     string `json:"before"`
	MergedOnly bool   `json:"merged"`
}

func NewDeleteResponse(err error) *DeleteResponse {
	status := make(map[string]*DeleteStatus)
	emsg := ""
	if err != nil {
		emsg = err.Error()
	}
	return &DeleteResponse{Error: err, ErrorMsg: emsg, Status: status}
}
func (dr *DeleteResponse) SetRepoBranchStatus(repository, branch string, deleted bool, err error) {
	emsg := ""
	if err != nil {
		emsg = err.Error()
	}
	var s map[string]bool
	if dr.Status[repository].Status == nil {
		s = make(map[string]bool)
	} else {
		s = dr.Status[repository].Status
	}
	s[branch] = deleted
	ds := &DeleteStatus{Status: s, Error: err, ErrorMsg: emsg}
	dr.Status[repository] = ds
}
func (dr *DeleteResponse) SetRepoStatus(repository string, status map[string]bool, err error) {
	emsg := ""
	if err != nil {
		emsg = err.Error()
	}
	ds := &DeleteStatus{Status: status, Error: err, ErrorMsg: emsg}
	dr.Status[repository] = ds
}

func (dr *DeleteResponse) SetError(err error) {
	dr.Error = err
	emsg := ""
	if err != nil {
		emsg = err.Error()
	}
	dr.ErrorMsg = emsg
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
				dr := NewDeleteResponse(fmt.Errorf("cannot convert branch %s to int. Error: %w", branch, err))
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
			dr := NewDeleteResponse(fmt.Errorf("cannot convert branch id %s to int. Error: %w", id, err))
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
	//status := make(map[string]*DeleteStatus)
	if managers == nil {
		dr := NewDeleteResponse(fmt.Errorf("no GitHub managers have been initialized yet"))
		reportError(w, r, dr, http.StatusAccepted)
		return
	}
	dr := NewDeleteResponse(nil)

	for _, m := range managers {
		err := m.GetClient().DeleteBranch(taskId)
		if err != nil {
			dr.SetRepoBranchStatus(m.Repository, branch, false, err)
			//status[m.Repository] = &DeleteStatus{Deleted: false, Error: err}
			misc.Debugf("failed to delete branch %s in repo %s. Error: %s", branch, m.Repository, err)
		} else {
			dr.SetRepoBranchStatus(m.Repository, branch, true, nil)
		}
	}
	marshalledStatus, err := json.Marshal(dr)
	w.WriteHeader(http.StatusAccepted)
	if err != nil {
		misc.Debugf("cannot marshal response %v. Error: %s", dr, err)
		w.Header().Add(misc.ContentTypeKey, misc.ContentTypeMarkdown)

		_, err := w.Write([]byte(fmt.Sprintf("cannot marshal response %v. Falling back to text. Error: %s", dr, err)))
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

func Cleanupbranches(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	cf := &CleanupForm{}
	err := dec.Decode(cf)
	if err != nil {
		misc.Debugf("could not parse json. Original message was: %v", r.Body)
		reportError(w, r, NewDeleteResponse(fmt.Errorf("could not parse json. Original message was: %v. Error: %w", r.Body, err)), http.StatusNotAcceptable)
		return
	}
	managers := github.GetAllManagers()
	if managers == nil {
		dr := NewDeleteResponse(fmt.Errorf("no GitHub managers have been initialized yet"))
		reportError(w, r, dr, http.StatusAccepted)
		return
	}
	var bt time.Time
	if cf.Before == "" {
		dr := NewDeleteResponse(fmt.Errorf("you have to pass before parameter in form of Unix time or RFC3339 date"))
		reportError(w, r, dr, http.StatusNotAcceptable)
		return
	} else {
		matchedUnix, err := regexp.MatchString("^[0-9]+$", cf.Before)
		if err != nil {
			reportError(w, r, NewDeleteResponse(fmt.Errorf("internal error. cannot compile regex for before time. Error: %w", err)), http.StatusInternalServerError)
			return
		}
		if matchedUnix {
			st, err := strconv.Atoi(cf.Before)

			if err != nil {
				reportError(w, r, NewDeleteResponse(fmt.Errorf("internal error. cannot convert unix before time %q to integer. Error: %w", cf.Before, err)), http.StatusNotAcceptable)
				return
			}
			bt = time.Unix(int64(st), 0)
		} else {
			bt, err = time.Parse(time.RFC1123, cf.Before)
			if err != nil {
				reportError(w, r, NewDeleteResponse(fmt.Errorf("cannot parse ISO RFC1123 date %q. Error: %w", cf.Before, err)), http.StatusNotAcceptable)
				return
			}
		}
	}
	dr := NewDeleteResponse(nil)

	for _, m := range managers {

		status, err := m.GetClient().CleanupBranches(&bt, cf.MergedOnly)
		if err != nil {
			dr.SetRepoStatus(m.Repository, status, err)
			//status[m.Repository] = &DeleteStatus{Deleted: false, Error: err}
			misc.Debugf("failed to cleanup branches in repo %s. Error: %s", m.Repository, err)
		} else {
			dr.SetRepoStatus(m.Repository, status, nil)
		}
	}
	marshalledStatus, err := json.Marshal(dr)
	w.WriteHeader(http.StatusAccepted)
	if err != nil {
		misc.Debugf("cannot marshal response %v. Error: %s", dr, err)
		w.Header().Add(misc.ContentTypeKey, misc.ContentTypeMarkdown)

		_, err := w.Write([]byte(fmt.Sprintf("cannot marshal response %v. Falling back to text. Error: %s", dr, err)))
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

func reportError(w http.ResponseWriter, r *http.Request, dr *DeleteResponse, status int) {
	marshalledStatus, err := json.Marshal(dr)
	w.WriteHeader(status)
	if err != nil {
		misc.Debugf("cannot marshal response %v. Error: %s", dr, err)
		w.Header().Add(misc.ContentTypeKey, misc.ContentTypeMarkdown)

		_, err := w.Write([]byte(fmt.Sprintf("cannot marshal response %v. Falling back to text. Error: %s", dr, err)))
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
	return
}
