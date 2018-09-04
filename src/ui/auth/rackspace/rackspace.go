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

	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/auth"
)

// Auth implements Authenticator interface to authenticate against Rackspace Managed Kubernetes Auth (kubernetes-auth)
type Auth struct {
	auth.DefaultAuthenticateHelper
	authURL string
	client  *http.Client
}

// Authenticate checks user's credential against the Rackspace Managed Kubernetes Auth (kubernetes-auth)
// if the check is successful a dummy record will be inserted into DB, such that this user can
// be associated to other entities in the system.
func (a *Auth) Authenticate(m models.AuthModel) (*models.User, error) {

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
	resp, err := a.client.Post(a.authURL+"/authenticate/token", "application/json", bytes.NewReader(authRequestBody))
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

	user, err := dao.GetUser(models.User{Username: authResp.Status.User.Username})
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

func (a *Auth) OnBoardUser(u *models.User) error {
	return nil
}

func (a *Auth) OnBoardGroup(g *models.UserGroup, altGroupName string) error {
	return errors.New("not implemented")
}

func (a *Auth) SearchUser(username string) (*models.User, error) {
	var queryCondition = models.User{
		Username: username,
	}

	return dao.GetUser(queryCondition)
}

func (a *Auth) SearchGroup(groupDN string) (*models.UserGroup, error) {
	return nil, errors.New("not implemented")
}

func (a *Auth) PostAuthenticate(u *models.User) error {
	return nil
}

var (
	rackspaceMK8SAuthURLTokenEndpoint string
)

func init() {
	a, err := setupAuth()
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Initializing Rackspace Managed Auth: url=%q", a.authURL)

	auth.Register("rackspace_mk8s_auth", a)
}

func setupAuth() (*Auth, error) {
	return &Auth{
		authURL: mk8sAuthURL(),
		client:  getClient(),
	}, nil
}

func getClient() *http.Client {
	const caPath = "/etc/openstack/certs/ca.pem"
	if needCustomCert(caPath) {
		ca, err := ioutil.ReadFile(caPath)
		if err != nil {
			log.Errorf("Error reading OpenStack CA Cert %s: %v", caPath, err)
			return http.DefaultClient
		}

		certs, err := x509.SystemCertPool()
		if err != nil {
			log.Errorf("Error getting cert pool: %v", err)
			return http.DefaultClient
		}

		certs.AppendCertsFromPEM(ca)

		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: certs,
				},
			},
		}
	}

	return http.DefaultClient
}

func needCustomCert(caPath string) bool {
	if !strings.HasPrefix(mk8sAuthURL(), "https") {
		return false
	}

	f, err := os.Stat(caPath)
	if err != nil {
		return false
	}

	if f.Size() == 0 {
		return false
	}

	return true
}

func mk8sAuthURL() string {
	const envVar = "RACKSPACE_MK8S_AUTH_URL"

	authURL := os.Getenv(envVar)

	if len(authURL) == 0 {
		log.Warningf("%s is not set", envVar)
		authURL = "http://app:8080"
	}

	if _, err := url.ParseRequestURI(authURL); err != nil {
		log.Fatalf("The env var %s is not a valid url %s", envVar, authURL)
	}

	return authURL
}

func randString() string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// emailAddress will return a unique email address for the given user
// Harbor requires email addresses in its database to be unique.
func emailAddress(u *models.User) string {
	// (fake) default email domain
	const defaultEmailDomain = "fake-rackspace-mk8s.com"

	if u.Email != "" {
		return u.Email
	}
	if u.Username != "" {
		return fmt.Sprintf("%s@%s", u.Username, defaultEmailDomain)
	}
	return fmt.Sprintf("%s@%s", randString(), defaultEmailDomain)
}
