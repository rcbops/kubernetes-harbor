# Harbor with Rackspace Managed Kubernetes Auth

This section explains how to create the development environment for developing the integration code for Harbor with Rackspace Managed Kubernetes Auth on Mac OSX. This uses your Docker/Docker Compose only because this is a local dev env. Integration with Kubernetes happens in [kubernetes-installer](https://github.com/rcbops/kubernetes-installer).

Read the [Harbor](#harbor) section. Because master is unstable, all development happens in a rackspace-mk8s-auth-release-X.X.X branch which is always based on an upstream stable release branch. The rackspace-mk8s-auth-release-X.X.X branch will need to be [rebased onto a new upstream stable release branch](#rebase-onto-a-new-upstream-stable-release-branch) as we upgrade to newer releases. This is akin to the "vendor branch" pattern in Git.

The only thing in the master branch is this section of the README just so it's easy to find.

## Dev Env

There are a few things you need to do to get a local dev env on Mac OSX going.

1. [Fork this repo](https://help.github.com/articles/fork-a-repo/) into your personal account.

1. Because Go expects source directories to exist in a set location, you will need to clone harbor to your `$GOPATH`, like so:

    ```bash
    mkdir -p $GOPATH/src/github.com/vmware
    git clone git@github.com:<my-github-username>/kubernetes-harbor.git $GOPATH/src/github.com/vmware/harbor
    cd $GOPATH/src/github.com/vmware/harbor

    # add remotes for both the upstream repo and the fork repo
    git remote add upstream git@github.com:vmware/harbor.git
    git remote add fork git@github.com:rcbops/kubernetes-harbor.git

    # list the remote branches, fetch the latest rackspace-mk8s-auth-release-X.X.X branch, and create a local tracking branch for it
    git ls-remote --heads fork
    git fetch fork rackspace-mk8s-auth-release-X.X.X
    git checkout --track fork/rackspace-mk8s-auth-release-X.X.X
    ```

1. Create a data dir.

    ```bash
    sudo mkdir /data
    sudo chown $(whoami):staff /data
    ```

    The `/data` dir persists all of the data that the Harbor apps use.

1. Open your Docker preferences and under File Sharing add dirs `/data` and `/var/log`.

## Run and Configure Kubernetes Auth

1. [Fork and clone the repo](https://help.github.com/articles/fork-a-repo/) [kubernetes-auth](https://github.com/rcbops/kubernetes-auth) into your personal account and clone it.

1. Run it with the dummy backend in the same network as Harbor.

    ```bash
    make run NETWORK_NAME_BASE=make_harbor
    ```

This command creates the following containers:

* a MySQL database for storing user info
* an auth service container for handling auth requests through an HTTP API

The auth service is set up using a dummy backend, meaning that it will not perform HTTP connections to a real backend like OpenStack Keystone, which is useful for testing. The result of this is that HTTP calls will always pass.

## Run Harbor

This section is cribbed from the [compile guide](docs/compile_guide.md). This deploys all Harbor components using your local Docker/Docker Compose. A few changes have been made to [harbor.cfg](make/harbor.cfg) to make this work out of the box.

```bash
make install GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage CLARITYIMAGE=vmware/harbor-clarity-ui-builder:1.2.7
```

This command creates all of the Harbor related containers. You can find out more about these containers by reading about the [Image Registry](https://github.com/rcbops/kubernetes-installer/blob/master/docs/support/TROUBLESHOOTING.md#image-registry) architecture.

## Login to Harbor as admin

1. Open [registry.127.0.0.1.nip.io](http://registry.127.0.0.1.nip.io) in your browser.

1. You can login using the username/password combo `admin/Harbor12345`. Becaue of the way Harbor is designed, the admin user is the one and only user that will use Harbor's own DB for authentication.

## Login to Harbor as a user

1. To login to Harbor as a user, you first need to create a user/token in the kubernetes-auth dummy backend.

1. Work through the [kubernetes-auth example](https://github.com/rcbops/kubernetes-auth#example) to create a user and token.

1. Open [registry.127.0.0.1.nip.io](http://registry.127.0.0.1.nip.io) in your browser.

1. The auth service will have returned a token as created in step 2. This token will be used as your Harbor password. It's important to note that since the Auth service has no concept of a Harbor username, you can choose any value when inputting your Harbor username - it will be ignored. The only identifier is the token returned by the Auth service

## Modify the Rackspace Managed Kubernetes Auth code in Harbor

1. The code which integrates Harbor with the Kubernetes Auth service lives in the `ui` component. If you would like to update this and re-deploy, you can use the following commands:

    ```bash
    git checkout -b my-feature-branch rackspace-mk8s-auth-release-X.X.X

    # Make your code changes.

    make redeploy_ui GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage
    ```

1. [Login to Harbor as a user](#login-to-harbor-as-a-user)

1. Check your code changes by tailing the ui.log.

    ```bash
    docker exec -it harbor-log tail -f /var/log/docker/$(date +%Y-%m-%d)/ui.log
    ```

1. Push your commit(s) and branch as usual.

    ```bash
    git commit -am "My feature"
    git push -u
    ```

1. When creating the pull request using the GitHub UI, take care to make sure you've selected the correct fork/branch pairs, like so.

    * `base fork:rcbops/kubernetes-harbor` `base: rackspace-mk8s-auth-release-X.X.X`
    * `head fork: <my-github-username>/kubernetes-harbor` `compare: my-feature-branch`

## Release a new image

1. Once a new feature or bugfix has been reviewed and merged, the Docker image needs to be rebuilt and pushed to [quay.io/rackspace/harbor-ui](https://quay.io/repository/rackspace/harbor-ui?tab=tags), like so:

    ```bash
    make compile_golangimage_ui GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage
    make -f make/photon/Makefile build_ui DOCKERIMAGENAME_UI=quay.io/rackspace/harbor-ui DEVFLAG=false
    docker push quay.io/rackspace/harbor-ui:$(git describe --tags)
    ```

1. Update the `REGISTRY_UI_IMAGE` var in [versions.sh](https://github.com/rcbops/kubernetes-installer/blob/master/hack/lib/versions.sh) of the kubernetes-installer to the result of `echo quay.io/rackspace/harbor-ui:$(git describe --tags)`.

## Rebase onto a new upstream stable release branch

When Harbor does a new release upstream, we may want to rebase our code onto that new upstream stable release branch. There are bound to be a number of conflicts to resolve when doing the `git merge` step.

⚠️ **The steps below are untested** since we haven't needed to do this kind of upgrade yet. Proceed slowly and with caution. Update these steps as necessary. ⚠️

```bash
git fetch upstream
git checkout --track upstream/release-1.3.0
git checkout -b rackspace-mk8s-auth-release-1.3.0 release-1.3.0
git merge rackspace-mk8s-auth-release-1.2.0
git push fork rackspace-mk8s-auth-release-1.2.0
```

# Harbor

[![Build Status](https://travis-ci.org/vmware/harbor.svg?branch=master)](https://travis-ci.org/vmware/harbor)
[![Coverage Status](https://coveralls.io/repos/github/vmware/harbor/badge.svg?branch=master)](https://coveralls.io/github/vmware/harbor?branch=master)

**Note**: The `master` branch may be in an *unstable or even broken state* during development.
Please use [releases](https://github.com/vmware/harbor/releases) instead of the `master` branch in order to get stable binaries.

<img alt="Harbor" src="docs/img/harbor_logo.png">

Project Harbor is an enterprise-class registry server that stores and distributes Docker images. Harbor extends the open source Docker Distribution by adding the functionalities usually required by an enterprise, such as security, identity and management. As an enterprise private registry, Harbor offers better performance and security. Having a registry closer to the build and run environment improves the image transfer efficiency. Harbor supports the setup of multiple registries and has images replicated between them. In addition, Harbor offers advanced security features, such as user management, access control and activity auditing.

### Features
* **Role based access control**: Users and repositories are organized via 'projects' and a user can have different permission for images under a project.
* **Policy based image replication**: Images can be replicated (synchronized) between multiple registry instances, with auto-retry on errors. Great for load balancing, high availability, multi-datacenter, hybrid and multi-cloud scenarios.
* **Vulnerability Scanning**: Harbor scans images regularly and warns users of vulnerabilities.
* **LDAP/AD support**: Harbor integrates with existing enterprise LDAP/AD for user authentication and management.
* **Image deletion & garbage collection**: Images can be deleted and their space can be recycled.
* **Notary**: Image authenticity can be ensured.
* **Graphical user portal**: User can easily browse, search repositories and manage projects.
* **Auditing**: All the operations to the repositories are tracked.
* **RESTful API**: RESTful APIs for most administrative operations, easy to integrate with external systems.
* **Easy deployment**: Provide both an online and offline installer.

### Install & Run

**System requirements:**

**On a Linux host:** docker 1.10.0+ and docker-compose 1.6.0+ .

Download binaries of **[Harbor release ](https://github.com/vmware/harbor/releases)** and follow **[Installation & Configuration Guide](docs/installation_guide.md)** to install Harbor.

Refer to **[User Guide](docs/user_guide.md)** for more details on how to use Harbor.

### Community
**Slack:** Join Harbor's community for discussion and ask questions: [VMware {code}](https://code.vmware.com/join/), Channel: #harbor.
**Email:** harbor@ vmware.com .
More info on [partners and users](partners.md).

### Contribution
We welcome contributions from the community. If you wish to contribute code and you have not signed our contributor license agreement (CLA), our bot will update the issue when you open a pull request. For any questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq). Contact us for any questions: harbor @vmware.com .

### License
Harbor is available under the [Apache 2 license](LICENSE).

This project uses open source components which have additional licensing terms.  The official docker images and licensing terms for these open source components can be found at the following locations:

* Photon OS 1.0: [docker image](https://hub.docker.com/_/photon/), [license](https://github.com/vmware/photon/blob/master/COPYING)
* MySQL 5.6: [docker image](https://hub.docker.com/_/mysql/), [license](https://github.com/docker-library/mysql/blob/master/LICENSE)

### Commercial Support
If you need commercial support of Harbor, please contact us for more information: harbor@ vmware.com .



