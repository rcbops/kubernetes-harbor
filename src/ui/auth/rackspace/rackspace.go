/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package rackspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"

	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/auth"
	"github.com/vmware/harbor/src/ui/config"
)

// Auth implements Authenticator interface to authenticate against Rackspace Managed Kubernetes Auth (kubernetes-auth)
type Auth struct{}

// AuthRequest is the request body format for kubernetes-auth
type AuthRequest struct {
	Spec struct {
		Token string `json:"token"`
	} `json:"spec"`
}

// AuthResponse is the response from kubernetes-auth
// If the user is authenticated, the User struct is filled. Otherwise, the Error string is filled
type AuthResponse struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Status     struct {
		Authenticated bool   `json:"authenticated"`
		Error         string `json:"error,omitempty"`
		User          struct {
			Username string              `json:"username"`
			UID      string              `json:"uid,omitempty"`
			Groups   []string            `json:"groups,omitempty"`
			Extra    map[string][]string `json:"extra,omitempty"`
		} `json:"user,omitempty"`
	} `json:"status"`
}

var (
	rackspaceMK8SAuthURLTokenEndpoint string
)

// Authenticate checks user's credential against the Rackspace Managed Kubernetes Auth (kubernetes-auth)
// if the check is successful a dummy record will be inserted into DB, such that this user can
// be associated to other entities in the system.
func (l *Auth) Authenticate(m models.AuthModel) (*models.User, error) {

	// kubernetes-auth only uses the token (m.Password) for auth. The username (m.Principal) isn't used at all.
	// In fact, a user could put anything at all into the username field. It must be ignored.
	// However, we log the username to help track the request because we can't put the token (m.Password) in the logs.
	log.Debugf("ProvidedUsername=%s Authentication attempt", m.Principal)

	// build auth request
	authRequest := &AuthRequest{}
	authRequest.Spec.Token = m.Password

	authRequestBody, err := json.Marshal(authRequest)
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error marshalling auth request: %v", m.Principal, err)
		return nil, err
	}

	log.Debugf("ProvidedUsername=%s Sending auth request: %s", m.Principal, rackspaceMK8SAuthURLTokenEndpoint)

	// send auth request
	resp, err := http.Post(rackspaceMK8SAuthURLTokenEndpoint, "application/json", bytes.NewReader(authRequestBody))
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error sending auth request: %v", m.Principal, err)
		return nil, err
	}
	defer resp.Body.Close()

	// read auth response body
	authRespBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error reading auth response: %v", m.Principal, err)
		return nil, err
	}

	// check for any status other than OK
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTPStatusCode=%d AuthResponseBody=%s", resp.StatusCode, authRespBody)
		log.Errorf("ProvidedUsername=%s Error non-200-OK status code on auth response: %s", m.Principal, errMsg)
		return nil, errors.New(errMsg)
	}

	// read auth response body as json
	authResp := AuthResponse{}
	err = json.Unmarshal([]byte(authRespBody), &authResp)
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error unmarshalling auth response: %v", m.Principal, err)
		return nil, err
	}

	log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s Authenticated=%t Getting user from database", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username, authResp.Status.Authenticated)

	user, err := dao.GetUser(models.User{Realname: authResp.Status.User.UID})
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error getting user from database: %v", m.Principal, err)
		return nil, err
	}

	// check if the user already exists in the database. if the user doesn't exist, create it.
	if user != nil {
		log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s exists in database", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username)

		// if the username changed in kubernetes-auth backend, update it in the database
		if user.Username != authResp.Status.User.Username {
			log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s backend username changed so updating database", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username)

			user.Username = authResp.Status.User.Username

			err = dao.ChangeUserProfile(*user)
			if err != nil {
				log.Errorf("ProvidedUsername=%s UID=%s BackendUsername=%s Error updating user profile: %v", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username, err)
				return nil, err
			}
		}
	} else {
		log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s does not exist in database so creating new user", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username)

		// set the Harbor Realname to the kubernetes-auth backend's UID because the UID is a static ID
		// whereas the kubernetes-auth backend's Username can change (so put it in the Harbor Username field for convenience)
		user = &models.User{}
		user.Realname = authResp.Status.User.UID
		user.Username = authResp.Status.User.Username
		user.Password = randString()
		user.Comment = "Do not edit this user"

		userID, err := dao.Register(*user)
		if err != nil {
			log.Errorf("ProvidedUsername=%s UID=%s BackendUsername=%s Error creating new user: %v", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username, err)
			return nil, err
		}

		user.UserID = int(userID)
	}

	return user, nil
}

func init() {
	auth.Register("rackspace_mk8s_auth", &Auth{})
	rackspaceMK8SAuthURL := config.RackspaceMK8SAuthURL()
	rackspaceMK8SAuthURLTokenEndpoint = rackspaceMK8SAuthURL + "/authenticate/token"
	log.Infof("Initializing Rackspace Managed Kubernetes Auth: rackspaceMK8SAuthURL=%s", rackspaceMK8SAuthURL)
}

func randString() string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
