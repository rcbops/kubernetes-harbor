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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/rcbops/kubernetes-harbor/src/common/dao"
	"github.com/rcbops/kubernetes-harbor/src/common/models"
	"github.com/rcbops/kubernetes-harbor/src/common/utils/log"
	"github.com/rcbops/kubernetes-harbor/src/ui/auth"
)

// Auth implements Authenticator interface to authenticate against Rackspace Managed Kubernetes Auth (kubernetes-auth)
type Auth struct {
	auth.DefaultAuthenticateHelper
}

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
	client                            http.Client
)

const openstackCACertFilePath = "/etc/openstack/certs/ca.pem"

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
	resp, err := client.Post(rackspaceMK8SAuthURLTokenEndpoint, "application/json", bytes.NewReader(authRequestBody))
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
			user.Email = emailAddress(user)

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
		// the Password field is required but unused so we set it to something random
		user = new(models.User)
		user.Realname = authResp.Status.User.UID
		user.Username = authResp.Status.User.Username
		user.Password = randString()
		user.Comment = "Do not edit this user"
		user.Email = emailAddress(user)

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
	log.Infof("Initialing Rackspace Kubernetes-as-a-Service Auth")

	auth.Register("rackspace_mk8s_auth", &Auth{})
	rackspaceMK8SAuthURL := mk8sAuthURL()
	rackspaceMK8SAuthURLTokenEndpoint = rackspaceMK8SAuthURL + "/authenticate/token"

	if strings.HasPrefix(rackspaceMK8SAuthURL, "https") {
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			log.Fatalf("failed to load system cert pool: %v", err)
		}

		// Load CA cert if present
		if _, err := os.Stat(openstackCACertFilePath); err == nil {
			caCert, err := ioutil.ReadFile(openstackCACertFilePath)
			if err != nil {
				log.Fatalf("error reading OpenStack CA Cert %s: %v", openstackCACertFilePath, err)
			}
			caCertPool.AppendCertsFromPEM(caCert)
		}

		// Setup HTTPS client
		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		client = http.Client{Transport: transport}
	} else {
		// Setup HTTP client
		client = http.Client{}
	}

	log.Infof("Initialized Rackspace Managed Kubernetes Auth: rackspaceMK8SAuthURL=%s", rackspaceMK8SAuthURL)
}

func mk8sAuthURL() string {
	rackspaceMK8SAuthURLEnvVar := "RACKSPACE_MK8S_AUTH_URL"

	rackspaceMK8SAuthURL := os.Getenv(rackspaceMK8SAuthURLEnvVar)

	if len(rackspaceMK8SAuthURL) == 0 {
		rackspaceMK8SAuthURL = "http://app:8080"
	}

	if _, err := url.ParseRequestURI(rackspaceMK8SAuthURL); err != nil {
		log.Fatalf("The env var %s is not a valid url %s", rackspaceMK8SAuthURLEnvVar, rackspaceMK8SAuthURL)
	}

	return rackspaceMK8SAuthURL
}

func randString() string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// (fake) default email domain
const defaultEmailDomain = "fake-rackspace-mk8s.com"

// emailAddress will return a unique email address for the given user
// Harbor requires email addresses in its database to be unique.
func emailAddress(u *models.User) string {
	if u.Email != "" {
		return u.Email
	}
	if u.Username != "" {
		return fmt.Sprintf("%s@%s", u.Username, defaultEmailDomain)
	}
	return fmt.Sprintf("%s@%s", randString(), defaultEmailDomain)
}
