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

	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/auth"
)

// Auth implements Authenticator interface to authenticate against kubernetes-auth
type Auth struct{}

// The response from Rackspace Managed Kubernetes Auth (kubernetes-auth)
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

// Authenticate checks user's credential against the Rackspace Managed Kubernetes Auth (kubernetes-auth)
// if the check is successful a dummy record will be inserted into DB, such that this user can
// be associated to other entities in the system.
func (l *Auth) Authenticate(m models.AuthModel) (*models.User, error) {
	authServer := "http://172.18.0.10:8080"

	// kubernetes-auth only uses the token (m.Password) for auth. The username (m.Principal) isn't used at all.
	// In fact, a user could put anything at all into the username field. It must be ignored.
	// However, we log the username to help track the request because we can't put the token in the logs.
	log.Debugf("Username=%s Authentication attempt", m.Principal)

	// build request
	authRequestBody := fmt.Sprintf("{\"spec\": {\"token\": \"%s\"}}", m.Password)
	req, err := http.NewRequest(http.MethodPost, authServer+"/authenticate/token", strings.NewReader(authRequestBody))
	if err != nil {
		log.Errorf("Username=%s Error=%v", m.Principal, err)
		return nil, err
	}

	// serd request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Username=%s Error=%v", m.Principal, err)
		return nil, err
	}
	defer resp.Body.Close()

	// read response body
	authRespBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Username=%s Error=%v", m.Principal, err)
		return nil, err
	}

	// check for server error
	if resp.StatusCode >= http.StatusInternalServerError {
		log.Errorf("Username=%s Error=%s", m.Principal, authRespBody)
		return nil, fmt.Errorf("%d %s", resp.StatusCode, authRespBody)
	}

	// read response body as json
	authResp := AuthResponse{}
	err = json.Unmarshal([]byte(authRespBody), &authResp)
	if err != nil {
		log.Errorf("Username=%s Error=%v", m.Principal, err)
		return nil, err
	}

	log.Debugf("Username=%s Authenticated=%t", m.Principal, authResp.Status.Authenticated)

	// check for any status other than OK
	if resp.StatusCode != http.StatusOK {
		log.Errorf("Username=%s Error=%s", m.Principal, authResp.Status.Error)
		return nil, fmt.Errorf("%d %s", resp.StatusCode, authResp.Status.Error)
	}

	u := models.User{}
	// set the Harbor Username to the kubernetes-auth backend's UID because the UID is a static ID
	// whereas the kubernetes-auth backend's Username can change (so put it in the comment for convenience)
	u.Username = authResp.Status.User.UID
	u.Comment = authResp.Status.User.Username

	return &u, nil
}

func init() {
	auth.Register("rackspace_mk8s_auth", &Auth{})
	// TODO: log auth server address here
}
