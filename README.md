# Harbor with Rackspace Managed Kubernetes Auth

This section explains how to create the development environment for developing the integration code for Harbor with Rackspace Managed Kubernetes Auth on Mac OSX. This uses your Docker/Docker Compose only because this is a local dev env. Integration with Kubernetes happens in [kubernetes-installer](https://github.com/rcbops/kubernetes-installer).

Read the [Harbor](#harbor) section. Because master is unstable, all development happens in the [rackspace-mk8s-auth branch](https://github.com/rcbops/kubernetes-harbor/tree/rackspace-mk8s-auth) which is always based on a stable release branch. The rackspace-mk8s-auth branch will need to be rebased onto newer Harbor stable branches as we upgrade to newer releases.

The only thing in the master branch is this section of the README just so it's easy to find.

## Dev Env

There are a few things you need to do to get a local dev env on Mac OSX going.

1. Fork the repo.

1. Even though the repo is git@github.com:rcbops/kubernetes-harbor.git it's easier to clone it to a directory structure where Go lang expects everything to be.

    ```bash
    mkdir $GOPATH/src/github.com/vmware
    git clone git@github.com:my-github-username/kubernetes-harbor.git $GOPATH/src/github.com/vmware/harbor
    cd $GOPATH/src/github.com/vmware/harbor
    git remote add upstream git@github.com:rcbops/kubernetes-harbor.git
    ```

1. Create a data dir.

    ```bash
    sudo mkdir /data
    sudo chown $(whoami):staff /data
    ```

1. Open your Docker preferences and under File Sharing add dirs `/data` and `/var/log`.

## Run and Configure Kubernetes Auth

1. Fork and clone [kubernetes-auth](https://github.com/rcbops/kubernetes-auth).

1. Run it with the dummy backend in the same network as Harbor.

    ```bash
    make run NETWORK_NAME_BASE=make_harbor
    ```

## Run Harbor

This is cribbed from the [compile guide](docs/compile_guide.md). This deploys all Harbor components using your local Docker/Docker Compose. A few changes have been made to [harbor.cfg](make/harbor.cfg) to make this work out of the box.

```bash
make install GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage CLARITYIMAGE=vmware/harbor-clarity-ui-builder:1.2.7
```

## Login to Harbor as admin

1. Open [registry.127.0.0.1.nip.io](http://registry.127.0.0.1.nip.io).

1. You can login using the username/password combo `admin/Harbor12345`. Becaue of the way Harbor is designed, the admin user is the one and only user that will use Harbor's own DB for authentication.

## Login to Harbor as a user

1. To login to Harbor as a user, you first need to create a user/token in the kubernetes-auth dummy backend.

1. Work through the [kubernetes-auth example](https://github.com/rcbops/kubernetes-auth#example) to create a user.

1. Open [registry.127.0.0.1.nip.io](http://registry.127.0.0.1.nip.io).

1. You can now login using the token as the password created in step 2. kubernetes-auth only uses the token for auth. The username isn't used at all. In fact, a user could put anything at all into the username field. It will be ignored.

## Modify the Rackspace Managed Kubernetes Auth code in Harbor

1. The auth code lives in the UI component. Additional targets were added to the [Makefile](Makefile) to make redeployment of the UI easy.

    ```bash
    git checkout -b my-feature-branch rackspace-mk8s-auth

    # Make your code changes.

    make redeploy_ui GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage
    ```

1. [Login to Harbor as a user](#login-to-harbor-as-a-user)

1. Check your code changes by tailing the ui.log.

    ```bash
    docker exec -it harbor-log tail -f /var/log/docker/$(date +%Y-%m-%d)/ui.log
    ```

## Release a new image

1. Compile the UI component, build the image, and push it to [quay.io/rackspace/harbor-ui](https://quay.io/repository/rackspace/harbor-ui?tab=tags).

    ```bash
    make compile_golangimage_ui GOBUILDIMAGE=golang:1.7.3 COMPILETAG=compile_golangimage
    make -f make/photon/Makefile build_ui DOCKERIMAGENAME_UI=quay.io/rackspace/harbor-ui DEVFLAG=false
    docker push quay.io/rackspace/harbor-ui:$(git describe --tags)
    ```

1. Update the `REGISTRY_UI_IMAGE` var in [versions.sh](https://github.com/rcbops/kubernetes-installer/blob/master/hack/lib/versions.sh) of the kubernetes-installer to the result of `echo quay.io/rackspace/harbor-ui:$(git describe --tags)`.

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



