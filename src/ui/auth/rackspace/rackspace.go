/*
   Copyright (c) 2016 VMware, Inc. All Rights Reserved.
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
	"fmt"
	"net/http"

	"github.com/vmware/harbor/src/adminserver/client"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/ui/auth"
)

// Auth implements Authenticator interface to authenticate against keystone
type Auth struct{}

var (
	// AuthClient to talk to kubernetes-auth
	AuthClient client.Client
)

// Authenticate checks user's credential against the Rackspace Managed Kubernetes Auth
// if the check is successful a dummy record will be inserted into DB, such that this user can
// be associated to other entities in the system.
func (l *Auth) Authenticate(m models.AuthModel) (*models.User, error) {
	log.Infof("authenticate rackspace managed kubernetes user: %s", m.Principal)

	return nil, fmt.Errorf("rackspace managed kubernetes unimplemented")
}

func init() {
	auth.Register("rackspace_mk8s_auth", &Auth{})
}
