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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/vmware/harbor/src/common"
	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/auth"
	"github.com/vmware/harbor/src/ui/config"
)

// Auth implements Authenticator interface to authenticate against kubernetes-auth
type Auth struct{}

// AuthResponse is the response from Rackspace Managed Kubernetes Auth (kubernetes-auth)
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
	authServerURLTokenEndpoint string
)

// Authenticate checks user's credential against the Rackspace Managed Kubernetes Auth (kubernetes-auth)
// if the check is successful a dummy record will be inserted into DB, such that this user can
// be associated to other entities in the system.
func (l *Auth) Authenticate(m models.AuthModel) (*models.User, error) {
	// kubernetes-auth only uses the token (m.Password) for auth. The username (m.Principal) isn't used at all.
	// In fact, a user could put anything at all into the username field. It must be ignored.
	// However, we log the username to help track the request because we can't put the token in the logs.
	log.Debugf("ProvidedUsername=%s Authentication attempt", m.Principal)

	// get the URL of the Rackspace Managed Kubernetes Auth service
	if authServerURLTokenEndpoint == "" {
		rackspaceMK8SAuthURL, err := config.RackspaceMK8SAuthURL()
		if err != nil {
			log.Errorf("ProvidedUsername=%s Problem getting value for %s Error=%v", m.Principal, common.RackspaceMK8SAuthURL, err)
			return nil, err
		}
		if rackspaceMK8SAuthURL == "" {
			log.Errorf("ProvidedUsername=%s Config value for %s was empty", m.Principal, common.RackspaceMK8SAuthURL)
			return nil, fmt.Errorf("Config value for %s was empty", common.RackspaceMK8SAuthURL)
		}

		authServerURLTokenEndpoint = rackspaceMK8SAuthURL + "/authenticate/token"
	}

	// build auth request
	authRequestBody := fmt.Sprintf("{\"spec\": {\"token\": \"%s\"}}", m.Password)
	req, err := http.NewRequest(http.MethodPost, authServerURLTokenEndpoint, strings.NewReader(authRequestBody))
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error=%v", m.Principal, err)
		return nil, err
	}

	log.Debugf("ProvidedUsername=%s Sending auth request to %s", m.Principal, authServerURLTokenEndpoint)

	// send auth request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error=%v", m.Principal, err)
		return nil, err
	}
	defer resp.Body.Close()

	// read auth response body
	authRespBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error=%v", m.Principal, err)
		return nil, err
	}

	// check for auth server error
	if resp.StatusCode >= http.StatusInternalServerError {
		log.Errorf("ProvidedUsername=%s Error=%s", m.Principal, authRespBody)
		return nil, fmt.Errorf("%d %s", resp.StatusCode, authRespBody)
	}

	// read auth response body as json
	authResp := AuthResponse{}
	err = json.Unmarshal([]byte(authRespBody), &authResp)
	if err != nil {
		log.Errorf("ProvidedUsername=%s Error=%v", m.Principal, err)
		return nil, err
	}

	log.Debugf("ProvidedUsername=%s Authenticated=%t", m.Principal, authResp.Status.Authenticated)

	// check for any status other than OK
	if resp.StatusCode != http.StatusOK {
		log.Errorf("ProvidedUsername=%s Error=%s", m.Principal, authResp.Status.Error)
		return nil, fmt.Errorf("%d %s", resp.StatusCode, authResp.Status.Error)
	}

	log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username)

	// check if the user already exists in the Harbor DB. if the user doesn't exist, create it.
	user := models.User{}
	// set the Harbor Realname to the kubernetes-auth backend's UID because the UID is a static ID
	// whereas the kubernetes-auth backend's Username can change (so put it in the Harbor Username field for convenience)
	user.Realname = authResp.Status.User.UID

	exist, err := dao.UserExists(user, "realname")
	if err != nil {
		log.Errorf("ProvidedUsername=%s UID=%s BackendUsername=%s Error=%v", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username, err)
		return nil, err
	}

	if exist {
		log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s exists in Harbor DB", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username)

		existingUser, err := dao.GetUser(user)
		if err != nil {
			log.Errorf("ProvidedUsername=%s UID=%s BackendUsername=%s Error=%v", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username, err)
			return nil, err
		}

		// if the username changed in kubernetes-auth backend, update it in the Harbor DB
		if existingUser.Username != authResp.Status.User.Username {
			log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s backend username change so updating Harbor DB", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username)

			user.Username = authResp.Status.User.Username
			existingUser.Username = authResp.Status.User.Username

			err = dao.ChangeUserProfile(user)
			if err != nil {
				log.Errorf("ProvidedUsername=%s UID=%s BackendUsername=%s Error=%v", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username, err)
				return nil, err
			}
		}

		user = *existingUser
	} else {
		log.Debugf("ProvidedUsername=%s UID=%s BackendUsername=%s does not exist in Harbor DB so creating new user", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username)

		user.Username = authResp.Status.User.Username
		user.Password = "ThisPassw0rdIsNotused"
		user.Comment = "Rackspace MK8S Auth DON'T EDIT"

		userID, err := dao.Register(user)
		if err != nil {
			log.Errorf("ProvidedUsername=%s UID=%s BackendUsername=%s Error=%v", m.Principal, authResp.Status.User.UID, authResp.Status.User.Username, err)
			return nil, err
		}
		user.UserID = int(userID)
	}

	return &user, nil
}

func init() {
	auth.Register("rackspace_mk8s_auth", &Auth{})
	log.Infof("Initializing Rackspace Managed Kubernetes Auth")
}
